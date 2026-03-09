import { useState, useEffect, useMemo } from 'react';
import { HelpSection, HelpSummary, HelpSubsection } from '../types';
import { getHelpSections, getHelpSection } from '../services/api';
import { aliasTemplates, triggerTemplates, variableTemplates } from '../templates';

// Order of help sections in the sidebar
const SECTION_ORDER = [
  'getting-started',
  'connecting',
  'terminal',
  'keybindings',
  'aliases',
  'triggers',
  'variables',
  'safety',
  'troubleshooting',
  'examples',
];

export default function HelpPage() {
  const [sections, setSections] = useState<HelpSummary[]>([]);
  const [activeSection, setActiveSection] = useState<string>('getting-started');
  const [activeContent, setActiveContent] = useState<HelpSection | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showExamples, setShowExamples] = useState(false);

  // Fetch all help sections on mount
  useEffect(() => {
    async function loadSections() {
      try {
        const data = await getHelpSections();
        // Sort sections according to SECTION_ORDER
        const sorted = data.sort((a, b) => {
          const aIndex = SECTION_ORDER.indexOf(a.slug);
          const bIndex = SECTION_ORDER.indexOf(b.slug);
          const aPos = aIndex >= 0 ? aIndex : 999;
          const bPos = bIndex >= 0 ? bIndex : 999;
          return aPos - bPos;
        });
        setSections(sorted);
      } catch (err) {
        console.error('Failed to load help sections:', err);
        setError('Failed to load help content');
      }
    }
    loadSections();
  }, []);

  // Fetch active section content when it changes
  useEffect(() => {
    async function loadContent() {
      if (!activeSection) return;
      
      // Handle examples section - no API call needed
      if (activeSection === 'examples') {
        setShowExamples(true);
        setLoading(false);
        setActiveContent(null);
        return;
      }
      
      setShowExamples(false);
      setLoading(true);
      try {
        const data = await getHelpSection(activeSection);
        setActiveContent(data);
      } catch (err) {
        console.error('Failed to load help content:', err);
        setError('Failed to load help section');
      } finally {
        setLoading(false);
      }
    }
    loadContent();
  }, [activeSection]);

  // Filter sections based on search query
  const filteredSections = useMemo(() => {
    if (!searchQuery.trim()) return sections;
    const query = searchQuery.toLowerCase();
    return sections.filter(
      (s) =>
        s.title.toLowerCase().includes(query) ||
        s.description.toLowerCase().includes(query)
    );
  }, [sections, searchQuery]);

  // Add examples section to filtered list when searching or not
  const navSections = useMemo(() => {
    const examplesSection: HelpSummary = {
      slug: 'examples',
      title: 'Examples',
      description: 'Ready-to-use automation templates',
    };
    
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      // Include examples in search results
      if ('examples'.includes(query) || 'template'.includes(query) || 'automation'.includes(query)) {
        return [...filteredSections, examplesSection];
      }
      return filteredSections;
    }
    return [...filteredSections, examplesSection];
  }, [sections, searchQuery, filteredSections]);

  // Get sections that match the search (for highlighting)
  const matchingSlugs = useMemo(() => {
    if (!searchQuery.trim()) return new Set<string>();
    const query = searchQuery.toLowerCase();
    const matching = new Set<string>();
    sections.forEach((s) => {
      if (
        s.title.toLowerCase().includes(query) ||
        s.description.toLowerCase().includes(query)
      ) {
        matching.add(s.slug);
      }
    });
    return matching;
  }, [sections, searchQuery]);

  // Render markdown-like content
  const renderContent = (content: string) => {
    // Split content into paragraphs
    const paragraphs = content.split('\n\n');
    
    return paragraphs.map((para, idx) => {
      // Handle headers
      if (para.startsWith('**') && para.includes(':**')) {
        const parts = para.split(':');
        const title = parts[0].replace(/\*\*/g, '');
        const rest = parts.slice(1).join(':');
        return (
          <div key={idx} style={{ marginBottom: '1rem' }}>
            <h4 style={{ marginBottom: '0.5rem', opacity: 0.9, fontWeight: 600 }}>
              {title}
            </h4>
            <div style={{ opacity: 0.8 }}>{renderInline(rest)}</div>
          </div>
        );
      }
      
      // Handle bullet lists
      if (para.includes('\n- ') || para.startsWith('- ')) {
        const lines = para.split('\n');
        const items = lines.map((line) => line.replace(/^-\s*/, ''));
        return (
          <ul key={idx} style={{ marginBottom: '1rem', paddingLeft: '1.5rem', opacity: 0.8 }}>
            {items.map((item, i) => (
              <li key={i} style={{ marginBottom: '0.25rem' }}>
                {renderInline(item)}
              </li>
            ))}
          </ul>
        );
      }
      
      // Handle tables (basic support)
      if (para.includes('|') && para.includes('---')) {
        const lines = para.split('\n').filter((l) => !l.includes('---') && l.trim());
        if (lines.length > 1) {
          const headers = lines[0].split('|').filter((c) => c.trim());
          const rows = lines.slice(1).map((line) =>
            line.split('|').filter((c) => c.trim())
          );
          return (
            <div key={idx} style={{ marginBottom: '1rem', overflowX: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.9rem' }}>
                <thead>
                  <tr>
                    {headers.map((h, i) => (
                      <th
                        key={i}
                        style={{
                          textAlign: 'left',
                          padding: '0.5rem',
                          borderBottom: '1px solid rgba(255,255,255,0.2)',
                          fontWeight: 600,
                        }}
                      >
                        {h.trim()}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row, ri) => (
                    <tr key={ri}>
                      {row.map((cell, ci) => (
                        <td
                          key={ci}
                          style={{
                            padding: '0.5rem',
                            borderBottom: '1px solid rgba(255,255,255,0.1)',
                          }}
                        >
                          {renderInline(cell)}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          );
        }
      }
      
      // Regular paragraph
      return (
        <p key={idx} style={{ marginBottom: '1rem', opacity: 0.8, lineHeight: 1.6 }}>
          {renderInline(para)}
        </p>
      );
    });
  };

  // Render inline formatting (bold, code, etc.)
  const renderInline = (text: string) => {
    // Split by ** for bold
    const parts = text.split(/(\*\*[^*]+\*\*)/g);
    return parts.map((part, idx) => {
      if (part.startsWith('**') && part.endsWith('**')) {
        return (
          <strong key={idx} style={{ fontWeight: 600 }}>
            {part.slice(2, -2)}
          </strong>
        );
      }
      // Code formatting
      if (part.startsWith('`') && part.endsWith('`')) {
        return (
          <code
            key={idx}
            style={{
              backgroundColor: 'rgba(255,255,255,0.1)',
              padding: '0.125rem 0.375rem',
              borderRadius: '0.25rem',
              fontSize: '0.875rem',
            }}
          >
            {part.slice(1, -1)}
          </code>
        );
      }
      return part;
    });
  };

  if (error && !sections.length) {
    return (
      <div style={{ padding: '2rem', textAlign: 'center' }}>
        <p style={{ color: '#ef4444' }}>{error}</p>
        <button
          onClick={() => window.location.reload()}
          style={{
            marginTop: '1rem',
            padding: '0.5rem 1rem',
            backgroundColor: '#3b82f6',
            color: 'white',
            border: 'none',
            borderRadius: '0.375rem',
            cursor: 'pointer',
          }}
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
      {/* Sidebar */}
      <aside
        style={{
          width: '280px',
          borderRight: '1px solid rgba(255,255,255,0.1)',
          display: 'flex',
          flexDirection: 'column',
          backgroundColor: 'rgba(0,0,0,0.2)',
          flexShrink: 0,
        }}
      >
        {/* Search */}
        <div style={{ padding: '1rem', borderBottom: '1px solid rgba(255,255,255,0.1)' }}>
          <div style={{ position: 'relative' }}>
            <input
              type="text"
              placeholder="Search help..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              style={{
                width: '100%',
                padding: '0.5rem 0.75rem 0.5rem 2.25rem',
                backgroundColor: 'rgba(255,255,255,0.1)',
                border: '1px solid rgba(255,255,255,0.2)',
                borderRadius: '0.375rem',
                color: 'white',
                fontSize: '0.875rem',
                outline: 'none',
                boxSizing: 'border-box',
              }}
            />
            <svg
              style={{
                position: 'absolute',
                left: '0.75rem',
                top: '50%',
                transform: 'translateY(-50%)',
                width: '1rem',
                height: '1rem',
                opacity: 0.5,
              }}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                style={{
                  position: 'absolute',
                  right: '0.5rem',
                  top: '50%',
                  transform: 'translateY(-50%)',
                  background: 'none',
                  border: 'none',
                  color: 'white',
                  opacity: 0.5,
                  cursor: 'pointer',
                  padding: '0.25rem',
                }}
              >
                ×
              </button>
            )}
          </div>
        </div>

        {/* Navigation */}
        <nav style={{ flex: 1, overflowY: 'auto', padding: '0.5rem' }}>
          {navSections.map((section) => (
            <button
              key={section.slug}
              onClick={() => {
                if (section.slug === 'examples') {
                  setShowExamples(true);
                  setActiveSection('examples');
                } else {
                  setShowExamples(false);
                  setActiveSection(section.slug);
                }
                setSearchQuery('');
              }}
              style={{
                display: 'block',
                width: '100%',
                padding: '0.75rem 1rem',
                backgroundColor:
                  activeSection === section.slug
                    ? 'rgba(59, 130, 246, 0.2)'
                    : 'transparent',
                border: 'none',
                borderRadius: '0.375rem',
                color: matchingSlugs.has(section.slug) ? '#60a5fa' : 'white',
                textAlign: 'left',
                cursor: 'pointer',
                fontSize: '0.875rem',
                marginBottom: '0.25rem',
                transition: 'background-color 0.15s',
              }}
            >
              <div style={{ fontWeight: 500 }}>{section.title}</div>
              {matchingSlugs.has(section.slug) && searchQuery && (
                <div style={{ fontSize: '0.75rem', opacity: 0.7, marginTop: '0.25rem' }}>
                  Matches: "{searchQuery}"
                </div>
              )}
            </button>
          ))}
          {navSections.length === 0 && searchQuery && (
            <div style={{ padding: '1rem', opacity: 0.6, fontSize: '0.875rem', textAlign: 'center' }}>
              No results found for "{searchQuery}"
            </div>
          )}
        </nav>
      </aside>

      {/* Content */}
      <main
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '2rem',
        }}
      >
        {loading ? (
          <div style={{ textAlign: 'center', padding: '2rem', opacity: 0.6 }}>
            Loading...
          </div>
        ) : showExamples ? (
          <div style={{ maxWidth: '1040px' }}>
            <h2
              style={{
                fontSize: '1.75rem',
                fontWeight: 600,
                marginBottom: '0.5rem',
              }}
            >
              Examples
            </h2>
            <p
              style={{
                opacity: 0.7,
                marginBottom: '2rem',
                fontSize: '1rem',
              }}
            >
              Ready-to-use automation templates. Click "Use Template" to add them to your automation.
            </p>

            {/* Alias Templates */}
            <div style={{ marginBottom: '2rem' }}>
              <h3
                style={{
                  fontSize: '1.25rem',
                  fontWeight: 600,
                  marginBottom: '1rem',
                  color: '#93c5fd',
                }}
              >
                Alias Templates
              </h3>
              <p style={{ opacity: 0.7, marginBottom: '1rem' }}>
                Aliases transform commands you type before they are sent to the server.
              </p>
              
              <div style={{ display: 'grid', gap: '0.75rem' }}>
                {aliasTemplates.map((template, index) => (
                  <div
                    key={index}
                    style={{
                      padding: '1rem',
                      backgroundColor: 'rgba(255,255,255,0.05)',
                      borderRadius: '0.5rem',
                      border: '1px solid rgba(255,255,255,0.1)',
                    }}
                  >
                    <div style={{ fontWeight: 600, marginBottom: '0.25rem' }}>
                      {template.name}
                    </div>
                    <div style={{ opacity: 0.7, fontSize: '0.875rem', marginBottom: '0.5rem' }}>
                      {template.description}
                    </div>
                    <div style={{ fontFamily: 'monospace', fontSize: '0.875rem' }}>
                      <code style={{ backgroundColor: 'rgba(0,0,0,0.3)', padding: '2px 6px', borderRadius: '3px' }}>
                        {template.pattern}
                      </code>
                      {' → '}
                      <code style={{ backgroundColor: 'rgba(0,0,0,0.3)', padding: '2px 6px', borderRadius: '3px' }}>
                        {template.replacement}
                      </code>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Trigger Templates */}
            <div style={{ marginBottom: '2rem' }}>
              <h3
                style={{
                  fontSize: '1.25rem',
                  fontWeight: 600,
                  marginBottom: '1rem',
                  color: '#93c5fd',
                }}
              >
                Trigger Templates
              </h3>
              <p style={{ opacity: 0.7, marginBottom: '1rem' }}>
                Triggers automatically respond when specific text appears in the game output.
              </p>
              
              <div style={{ display: 'grid', gap: '0.75rem' }}>
                {triggerTemplates.map((template, index) => (
                  <div
                    key={index}
                    style={{
                      padding: '1rem',
                      backgroundColor: 'rgba(255,255,255,0.05)',
                      borderRadius: '0.5rem',
                      border: '1px solid rgba(255,255,255,0.1)',
                    }}
                  >
                    <div style={{ fontWeight: 600, marginBottom: '0.25rem' }}>
                      {template.name}
                    </div>
                    <div style={{ opacity: 0.7, fontSize: '0.875rem', marginBottom: '0.5rem' }}>
                      {template.description}
                    </div>
                    <div style={{ fontFamily: 'monospace', fontSize: '0.875rem' }}>
                      <span style={{ opacity: 0.7 }}>When: </span>
                      <code style={{ backgroundColor: 'rgba(0,0,0,0.3)', padding: '2px 6px', borderRadius: '3px' }}>
                        {template.match}
                      </code>
                      {' → '}
                      <code style={{ backgroundColor: 'rgba(0,0,0,0.3)', padding: '2px 6px', borderRadius: '3px' }}>
                        {template.action}
                      </code>
                      <span style={{ opacity: 0.5, fontSize: '0.75rem', marginLeft: '0.5rem' }}>
                        (cooldown: {template.cooldown_ms}ms)
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Variable Templates */}
            <div style={{ marginBottom: '2rem' }}>
              <h3
                style={{
                  fontSize: '1.25rem',
                  fontWeight: 600,
                  marginBottom: '1rem',
                  color: '#93c5fd',
                }}
              >
                Variable Templates
              </h3>
              <p style={{ opacity: 0.7, marginBottom: '1rem' }}>
                Variables store reusable values that can be used in aliases and triggers.
              </p>
              
              <div style={{ display: 'grid', gap: '0.75rem' }}>
                {variableTemplates.map((template, index) => (
                  <div
                    key={index}
                    style={{
                      padding: '1rem',
                      backgroundColor: 'rgba(255,255,255,0.05)',
                      borderRadius: '0.5rem',
                      border: '1px solid rgba(255,255,255,0.1)',
                    }}
                  >
                    <div style={{ fontWeight: 600, marginBottom: '0.25rem' }}>
                      {template.name}
                    </div>
                    <div style={{ opacity: 0.7, fontSize: '0.875rem', marginBottom: '0.5rem' }}>
                      {template.description}
                    </div>
                    <div style={{ fontFamily: 'monospace', fontSize: '0.875rem' }}>
                      <span style={{ opacity: 0.7 }}>Value: </span>
                      <code style={{ backgroundColor: 'rgba(0,0,0,0.3)', padding: '2px 6px', borderRadius: '3px' }}>
                        {template.value || '(empty)'}
                      </code>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        ) : activeContent ? (
          <div style={{ maxWidth: '1040px' }}>
            <h2
              style={{
                fontSize: '1.75rem',
                fontWeight: 600,
                marginBottom: '0.5rem',
              }}
            >
              {activeContent.title}
            </h2>
            <p
              style={{
                opacity: 0.7,
                marginBottom: '2rem',
                fontSize: '1rem',
              }}
            >
              {activeContent.description}
            </p>

            {activeContent.sections.map((subsection: HelpSubsection, idx: number) => (
              <div
                key={idx}
                style={{
                  marginBottom: '2rem',
                  paddingBottom: '2rem',
                  borderBottom:
                    idx < activeContent.sections.length - 1
                      ? '1px solid rgba(255,255,255,0.1)'
                      : 'none',
                }}
              >
                <h3
                  style={{
                    fontSize: '1.25rem',
                    fontWeight: 600,
                    marginBottom: '1rem',
                    color: '#93c5fd',
                  }}
                >
                  {subsection.title}
                </h3>
                {renderContent(subsection.content)}
              </div>
            ))}
          </div>
        ) : (
          <div style={{ textAlign: 'center', padding: '2rem', opacity: 0.6 }}>
            Select a topic from the sidebar to view help content
          </div>
        )}
      </main>
    </div>
  );
}
