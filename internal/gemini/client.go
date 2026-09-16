// Package gemini is a hand-written net/http client for the Gemini
// generateContent REST endpoint (D-18: no SDK, no new go.mod entry). It
// constrains the model's answer to a {reasoning, command} JSON schema so
// the caller always gets back one short plain-language "why" and exactly
// one game command (D-10), or a typed, named failure (D-13).
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Answer is the shape the model is constrained to produce: a short
// plain-language reasoning for the owner, plus exactly one game command
// (D-10). It is returned unmodified by GenerateContent — this package does
// not trim, rewrite or validate Command. Command validation is a security
// boundary that belongs in the driver, in front of the ICM dispatcher
// (plan 03-08, T-3-01), so that check exists in exactly one place and
// cannot be half-done in two.
type Answer struct {
	Reasoning string `json:"reasoning"`
	Command   string `json:"command"`
}

// Kind names the shape of a GenerateContent failure so a caller can branch
// on it without a type switch. These five kinds are what plan 03-08 maps
// onto D-13's failure notices; they are deliberately not collapsed into a
// single generic error.
type Kind string

const (
	// KindTransport covers network/context errors and any non-200 status
	// this package does not otherwise recognize.
	KindTransport Kind = "transport"
	// KindBadRequest is a Gemini HTTP 400 — a malformed request body or an
	// invalid generationConfig.
	KindBadRequest Kind = "bad_request"
	// KindAuth is a Gemini HTTP 401 or 403 — the API key is missing,
	// invalid, expired, or lacks permission for the model.
	KindAuth Kind = "auth"
	// KindRateLimited is a Gemini HTTP 429 (RESOURCE_EXHAUSTED) — quota or
	// rate limit exceeded (D-19).
	KindRateLimited Kind = "rate_limited"
	// KindMalformed covers an empty candidates list, an empty parts list,
	// or an inner answer that fails the second JSON decode (RESEARCH
	// Pitfall 5).
	KindMalformed Kind = "malformed"
)

// Error is the typed error every GenerateContent failure comes back as.
// Status is the HTTP status Gemini returned, or 0 for a transport error
// that never reached the vendor.
type Error struct {
	Kind    Kind
	Status  int
	Message string
}

func (e *Error) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("gemini: %s (status %d): %s", e.Kind, e.Status, e.Message)
	}
	return fmt.Sprintf("gemini: %s: %s", e.Kind, e.Message)
}

// ErrorKind returns the Kind of err for callers that do not want a type
// switch. Anything that is not a *Error (including nil) is reported as
// KindTransport.
func ErrorKind(err error) Kind {
	if gerr, ok := err.(*Error); ok {
		return gerr.Kind
	}
	return KindTransport
}

// Client is a Gemini generateContent client.
type Client struct {
	httpClient *http.Client
}

// NewClient returns a Client with the given request timeout. A zero
// timeout defaults to 30 seconds.
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{httpClient: &http.Client{Timeout: timeout}}
}

// contentPart, contentBlock, responseSchema and generationConfig mirror
// the Gemini generateContent request body exactly as documented at
// ai.google.dev/api/generate-content (RESEARCH.md "Gemini API Reference").
type contentPart struct {
	Text string `json:"text"`
}

type contentBlock struct {
	Role  string        `json:"role,omitempty"`
	Parts []contentPart `json:"parts"`
}

type schemaProperty struct {
	Type string `json:"type"`
}

type responseSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]schemaProperty `json:"properties"`
	Required   []string                  `json:"required"`
	// PropertyOrdering tells Gemini's structured-output decoder the order
	// in which to generate the named properties (the Gemini API's own
	// generation order follows this list, not Go's unordered map
	// iteration over Properties). ReviewCommand sets this so the model
	// writes its reason before its blocked boolean — reasoning-then-decide
	// rather than decide-then-rationalize. GenerateContent leaves it nil;
	// omitempty keeps that request body unchanged.
	PropertyOrdering []string `json:"propertyOrdering,omitempty"`
}

type generationConfig struct {
	ResponseMimeType string         `json:"responseMimeType"`
	ResponseSchema   responseSchema `json:"responseSchema"`
}

type generateContentRequest struct {
	SystemInstruction contentBlock     `json:"systemInstruction"`
	Contents          []contentBlock   `json:"contents"`
	GenerationConfig  generationConfig `json:"generationConfig"`
}

// generateContentResponse is the outer HTTP response envelope. The inner
// Text field is itself a JSON string (the model's structured answer) and
// must be decoded a second time into Answer — skipping that double decode
// shows the owner raw JSON where the reasoning should be (RESEARCH
// Pitfall 5).
type generateContentResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// vendorErrorBody is Gemini's error envelope shape:
// {"error": {"code": 429, "message": "...", "status": "RESOURCE_EXHAUSTED"}}
type vendorErrorBody struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// maxErrorBodyBytes bounds how much of a non-200 response body this
// package will read, so a misbehaving or malicious endpoint cannot make
// error handling itself unbounded.
const maxErrorBodyBytes = 4096

// maxSuccessBodyBytes bounds a 200 response body.
const maxSuccessBodyBytes = 1 << 20 // 1MB

// GenerateContent sends one request to Gemini's generateContent endpoint
// and returns the model's {reasoning, command} answer. endpoint and model
// always arrive as parameters — this package never hard-codes a vendor
// host or a model identifier (CON-model-config): the Gemini model lineup
// moves within this project's own timeline (RESEARCH, State of the Art).
//
// Nothing about the outbound request — URL, headers, key, prompt, or
// answer — is logged here; plan 03-08 logs the outcome with ids and
// lengths only, never the content of any of these.
func (c *Client) GenerateContent(ctx context.Context, endpoint, model, apiKey, systemInstruction, userText string) (*Answer, error) {
	schema := responseSchema{
		Type: "object",
		Properties: map[string]schemaProperty{
			"reasoning": {Type: "string"},
			"command":   {Type: "string"},
		},
		Required: []string{"reasoning", "command"},
	}
	inner, err := c.doGenerate(ctx, endpoint, model, apiKey, systemInstruction, userText, schema)
	if err != nil {
		return nil, err
	}
	var answer Answer
	if err := json.Unmarshal(inner, &answer); err != nil {
		return nil, &Error{Kind: KindMalformed, Message: "could not decode the model's structured answer"}
	}
	return &answer, nil
}

// doGenerate is the shared plumbing behind GenerateContent and
// ReviewCommand: it marshals a request carrying the given responseSchema,
// posts it with the key in its one dedicated header (never a URL query
// parameter — a URL with the key embedded in it lands in every access log,
// proxy log, and Go transport error string that records the request URL,
// the exact class of problem DR-2-01 exists to close, RESEARCH Pitfall 3),
// maps a non-200 response through errorFromResponse, decodes the outer
// envelope, and returns the first candidate's first part as raw JSON bytes
// for the caller to decode a second time into its own answer shape.
// Nothing about the outbound request or the response is logged here.
func (c *Client) doGenerate(ctx context.Context, endpoint, model, apiKey, systemInstruction, userText string, schema responseSchema) ([]byte, error) {
	reqBody := generateContentRequest{
		SystemInstruction: contentBlock{Parts: []contentPart{{Text: systemInstruction}}},
		Contents: []contentBlock{
			{Role: "user", Parts: []contentPart{{Text: userText}}},
		},
		GenerationConfig: generationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema:   schema,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &Error{Kind: KindTransport, Message: fmt.Sprintf("marshal request: %v", err)}
	}

	url := strings.TrimRight(endpoint, "/") + "/v1beta/models/" + model + ":generateContent"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, &Error{Kind: KindTransport, Message: fmt.Sprintf("build request: %v", err)}
	}
	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &Error{Kind: KindTransport, Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errorFromResponse(resp)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSuccessBodyBytes))
	if err != nil {
		return nil, &Error{Kind: KindTransport, Message: err.Error()}
	}

	var envelope generateContentResponse
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, &Error{Kind: KindMalformed, Message: "could not decode Gemini response envelope"}
	}
	if len(envelope.Candidates) == 0 || len(envelope.Candidates[0].Content.Parts) == 0 {
		return nil, &Error{Kind: KindMalformed, Message: "Gemini returned no candidates"}
	}

	return []byte(envelope.Candidates[0].Content.Parts[0].Text), nil
}

// ReviewAnswer is the reviewer's constrained verdict on a command another
// model already chose: whether to stop it, plus a one-sentence reason
// written for the owner (D-03, D-08). This package assigns no meaning to
// Blocked or Reason beyond decoding them — deciding what a verdict means,
// what to do about it, and what the owner is told all live in the driver.
type ReviewAnswer struct {
	Blocked bool   `json:"blocked"`
	Reason  string `json:"reason"`
}

// ReviewCommand asks the model to judge an already-chosen command (D-03).
// Its error taxonomy, its key-in-header discipline, and its behaviour of
// never repeating a failed call are identical to GenerateContent's — see
// that method's doc comment, which applies verbatim here. This package
// judges nothing about what the returned verdict means.
func (c *Client) ReviewCommand(ctx context.Context, endpoint, model, apiKey, systemInstruction, userText string) (*ReviewAnswer, error) {
	// PropertyOrdering places "reason" ahead of "blocked" so Gemini's
	// structured-output generation writes the reasoning text first and the
	// boolean verdict second, forcing the review to reason before it
	// decides rather than decide and then backfill a justification.
	schema := responseSchema{
		Type: "object",
		Properties: map[string]schemaProperty{
			"blocked": {Type: "boolean"},
			"reason":  {Type: "string"},
		},
		Required:         []string{"blocked", "reason"},
		PropertyOrdering: []string{"reason", "blocked"},
	}
	inner, err := c.doGenerate(ctx, endpoint, model, apiKey, systemInstruction, userText, schema)
	if err != nil {
		return nil, err
	}
	var answer ReviewAnswer
	if err := json.Unmarshal(inner, &answer); err != nil {
		return nil, &Error{Kind: KindMalformed, Message: "could not decode the reviewer's structured answer"}
	}
	return &answer, nil
}

// errorFromResponse maps a non-200 Gemini response to a typed *Error,
// reading at most maxErrorBodyBytes of the body and recording only the
// vendor's error.message when it parses — never the request body and
// never any header.
func errorFromResponse(resp *http.Response) *Error {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))

	var kind Kind
	switch resp.StatusCode {
	case http.StatusBadRequest:
		kind = KindBadRequest
	case http.StatusUnauthorized, http.StatusForbidden:
		kind = KindAuth
	case http.StatusTooManyRequests:
		kind = KindRateLimited
	default:
		kind = KindTransport
	}

	message := fmt.Sprintf("Gemini returned HTTP %d", resp.StatusCode)
	var vendorErr vendorErrorBody
	if json.Unmarshal(data, &vendorErr) == nil && vendorErr.Error.Message != "" {
		message = vendorErr.Error.Message
	}

	return &Error{Kind: kind, Status: resp.StatusCode, Message: message}
}
