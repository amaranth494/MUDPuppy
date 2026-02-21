// MUDPuppy Authentication App

const API_BASE = '/api/v1';

// Application State
const state = {
    currentUser: null,
    isLoading: false,
    currentView: 'login', // 'login', 'register', 'otp', 'dashboard'
    pendingEmail: null
};

// DOM Elements
const elements = {
    loginSection: null,
    registerSection: null,
    otpSection: null,
    dashboardSection: null,
    loginForm: null,
    registerForm: null,
    otpForm: null,
    loginEmail: null,
    registerEmail: null,
    otpInput: null,
    userEmail: null,
    logoutBtn: null,
    loginMessage: null,
    registerMessage: null,
    otpMessage: null
};

// Initialize the application
function init() {
    cacheElements();
    bindEvents();
    checkAuthState();
}

// Cache DOM elements
function cacheElements() {
    elements.loginSection = document.getElementById('login-section');
    elements.registerSection = document.getElementById('register-section');
    elements.otpSection = document.getElementById('otp-section');
    elements.dashboardSection = document.getElementById('dashboard-section');
    
    // Login forms (two-step)
    elements.loginEmailForm = document.getElementById('login-email-form');
    elements.loginOtpForm = document.getElementById('login-otp-form');
    
    // Register form
    elements.registerForm = document.getElementById('register-form');
    
    // OTP form (register flow)
    elements.otpForm = document.getElementById('otp-form');
    
    // Input fields
    elements.loginEmail = document.getElementById('login-email');
    elements.registerEmail = document.getElementById('register-email');
    elements.loginOtpInput = document.getElementById('login-otp-input');
    elements.otpInput = document.getElementById('otp-input') || document.getElementById('otp-input-verify');
    elements.verifyOtpInput = document.getElementById('otp-input-verify');
    elements.userEmail = document.getElementById('user-email');
    elements.logoutBtn = document.getElementById('logout-btn');
    
    // Messages
    elements.loginMessage = document.getElementById('login-message');
    elements.registerMessage = document.getElementById('register-message');
    elements.otpMessage = document.getElementById('otp-message');
    
    // Login steps
    elements.loginEmailStep = document.getElementById('login-email-step');
    elements.loginOtpStep = document.getElementById('login-otp-step');
    
    // State for OTP inputs
    state.loginOtpValue = '';
    state.verifyOtpValue = '';
}

// Bind event listeners
function bindEvents() {
    // Login forms (two-step)
    elements.loginEmailForm?.addEventListener('submit', handleLoginEmailSubmit);
    elements.loginOtpForm?.addEventListener('submit', handleLoginOtpSubmit);
    
    // Register form
    elements.registerForm?.addEventListener('submit', handleRegisterSubmit);
    
    // OTP form
    elements.otpForm?.addEventListener('submit', handleOTPSubmit);
    
    // Logout button
    elements.logoutBtn?.addEventListener('click', handleLogout);
    
    // Link navigation
    document.getElementById('show-register')?.addEventListener('click', (e) => {
        e.preventDefault();
        showView('register');
    });
    
    document.getElementById('show-login')?.addEventListener('click', (e) => {
        e.preventDefault();
        showView('login');
    });
    
    document.getElementById('resend-otp')?.addEventListener('click', (e) => {
        e.preventDefault();
        resendOTP();
    });
    
    document.getElementById('back-to-register')?.addEventListener('click', (e) => {
        e.preventDefault();
        showView('register');
    });
    
    // Login page links
    document.getElementById('login-resend-otp')?.addEventListener('click', (e) => {
        e.preventDefault();
        loginResendOTP();
    });
    
    document.getElementById('login-back-to-email')?.addEventListener('click', (e) => {
        e.preventDefault();
        showLoginEmailStep();
    });
}

// Check authentication state on load
async function checkAuthState() {
    try {
        const response = await fetch(`${API_BASE}/me`, {
            credentials: 'include'
        });
        
        if (response.ok) {
            const user = await response.json();
            state.currentUser = user;
            showDashboard(user);
        } else {
            showView('login');
        }
    } catch (error) {
        console.error('Auth check failed:', error);
        showView('login');
    }
}

// Show a specific view
function showView(view) {
    state.currentView = view;
    
    // Hide all sections
    elements.loginSection?.classList.remove('active');
    elements.registerSection?.classList.remove('active');
    elements.otpSection?.classList.remove('active');
    elements.dashboardSection?.classList.remove('active');
    
    // Show selected section
    switch (view) {
        case 'login':
            elements.loginSection?.classList.add('active');
            clearMessages();
            showLoginEmailStep(); // Reset to email step
            elements.loginEmail?.focus();
            break;
        case 'register':
            elements.registerSection?.classList.add('active');
            clearMessages();
            elements.registerEmail?.focus();
            break;
        case 'otp':
            elements.otpSection?.classList.add('active');
            clearMessages();
            elements.otpInput?.focus();
            break;
        case 'dashboard':
            elements.dashboardSection?.classList.add('active');
            break;
    }
}

// Show dashboard with user info
function showDashboard(user) {
    if (elements.userEmail) {
        elements.userEmail.textContent = user.email;
    }
    showView('dashboard');
}

// Clear all messages
function clearMessages() {
    if (elements.loginMessage) {
        elements.loginMessage.textContent = '';
        elements.loginMessage.className = 'message';
        elements.loginMessage.style.display = 'none';
    }
    if (elements.registerMessage) {
        elements.registerMessage.textContent = '';
        elements.registerMessage.className = 'message';
        elements.registerMessage.style.display = 'none';
    }
    if (elements.otpMessage) {
        elements.otpMessage.textContent = '';
        elements.otpMessage.className = 'message';
        elements.otpMessage.style.display = 'none';
    }
}

// Show message helper
function showMessage(element, message, type = 'error') {
    if (!element) return;
    element.textContent = message;
    element.className = `message message-${type}`;
    element.style.display = 'block';
}

// Handle login email submission (Step 1: send OTP)
async function handleLoginEmailSubmit(e) {
    e.preventDefault();
    
    const email = elements.loginEmail?.value?.trim();
    
    if (!email || !email.includes('@')) {
        showMessage(elements.loginMessage, 'Please enter a valid email address', 'error');
        return;
    }
    
    setLoading(elements.loginEmailForm, true);
    
    try {
        // Call /api/v1/send-otp to send OTP (requires existing user)
        const response = await fetch(`${API_BASE}/send-otp`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            credentials: 'include',
            body: JSON.stringify({ email })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            // OTP sent successfully - show OTP step
            state.pendingEmail = email;
            showLoginOtpStep();
            showMessage(elements.loginMessage, 'Verification code sent! Check your email.', 'success');
        } else if (response.status === 429) {
            showMessage(elements.loginMessage, data.error || 'Too many requests. Please try again later.', 'error');
        } else if (response.status === 404) {
            // User not found - prompt to register
            showMessage(elements.loginMessage, 'Email not found. Please register first.', 'error');
        } else {
            showMessage(elements.loginMessage, data.error || 'Failed to send code. Please try again.', 'error');
        }
    } catch (error) {
        console.error('Send OTP error:', error);
        showMessage(elements.loginMessage, 'Network error. Please try again.', 'error');
    } finally {
        setLoading(elements.loginEmailForm, false);
    }
}

// Handle login OTP verification (Step 2: verify OTP and login)
async function handleLoginOtpSubmit(e) {
    e.preventDefault();
    
    const email = state.pendingEmail;
    const otp = elements.loginOtpInput?.value?.trim();
    
    if (!email || !otp) {
        showMessage(elements.loginMessage, 'Please enter the verification code', 'error');
        return;
    }
    
    setLoading(elements.loginOtpForm, true);
    
    try {
        const response = await fetch(`${API_BASE}/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            credentials: 'include',
            body: JSON.stringify({ email, otp })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            // Login successful - fetch user info
            const userResponse = await fetch(`${API_BASE}/me`, {
                credentials: 'include'
            });
            
            if (userResponse.ok) {
                const user = await userResponse.json();
                state.currentUser = user;
                state.pendingEmail = null;
                showDashboard(user);
            } else {
                showMessage(elements.loginMessage, 'Login successful but could not fetch user info', 'error');
            }
        } else {
            showMessage(elements.loginMessage, data.error || 'Invalid or expired verification code', 'error');
        }
    } catch (error) {
        console.error('Login error:', error);
        // Check if this was actually an HTTP error response
        showMessage(elements.loginMessage, 'Invalid or expired verification code. Please request a new one.', 'error');
    } finally {
        setLoading(elements.loginOtpForm, false);
    }
}

// Show the login email step (Step 1)
function showLoginEmailStep() {
    elements.loginEmailStep.style.display = 'block';
    elements.loginOtpStep.style.display = 'none';
    state.pendingEmail = null;
    clearMessages();
}

// Show the login OTP step (Step 2)
function showLoginOtpStep() {
    elements.loginEmailStep.style.display = 'none';
    elements.loginOtpStep.style.display = 'block';
    if (elements.loginOtpInput) {
        elements.loginOtpInput.value = '';
        elements.loginOtpInput.focus();
    }
}

// Resend OTP for login
async function loginResendOTP() {
    const email = state.pendingEmail;
    
    if (!email) {
        showMessage(elements.loginMessage, 'No email address found. Please go back to email entry.', 'error');
        return;
    }
    
    setLoading(elements.loginOtpForm, true);
    
    try {
        const response = await fetch(`${API_BASE}/send-otp`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            credentials: 'include',
            body: JSON.stringify({ email })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            showMessage(elements.loginMessage, 'New verification code sent!', 'success');
        } else if (response.status === 429) {
            showMessage(elements.loginMessage, data.error || 'Too many requests. Please try again later.', 'error');
        } else if (response.status === 404) {
            showMessage(elements.loginMessage, 'Email not found. Please register first.', 'error');
        } else {
            showMessage(elements.loginMessage, data.error || 'Failed to resend code.', 'error');
        }
    } catch (error) {
        console.error('Resend OTP error:', error);
        showMessage(elements.loginMessage, 'Network error. Please try again.', 'error');
    } finally {
        setLoading(elements.loginOtpForm, false);
    }
}

// Handle registration form submission
async function handleRegisterSubmit(e) {
    e.preventDefault();
    
    const email = elements.registerEmail?.value?.trim();
    
    if (!email || !email.includes('@')) {
        showMessage(elements.registerMessage, 'Please enter a valid email address', 'error');
        return;
    }
    
    setLoading(elements.registerForm, true);
    
    try {
        const response = await fetch(`${API_BASE}/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            credentials: 'include',
            body: JSON.stringify({ email })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            // Store email for OTP verification and show OTP section
            state.pendingEmail = email;
            showMessage(elements.otpMessage, 'Verification code sent! Check your email.', 'success');
            showView('otp');
            
            // Show email in OTP section
            const emailDisplay = document.getElementById('otp-email-value');
            if (emailDisplay) {
                emailDisplay.textContent = email;
            }
            
            // Clear OTP input and focus
            if (elements.otpInput) {
                elements.otpInput.value = '';
                elements.otpInput.focus();
            }
        } else if (response.status === 429) {
            showMessage(elements.registerMessage, data.error || 'Too many requests. Please try again later.', 'error');
        } else {
            showMessage(elements.registerMessage, data.error || 'Registration failed', 'error');
        }
    } catch (error) {
        console.error('Registration error:', error);
        showMessage(elements.registerMessage, 'Network error. Please try again.', 'error');
    } finally {
        setLoading(elements.registerForm, false);
    }
}

// Handle OTP form submission
async function handleOTPSubmit(e) {
    e.preventDefault();
    
    const email = state.pendingEmail || elements.registerEmail?.value?.trim();
    const otp = elements.verifyOtpInput?.value?.trim();
    
    if (!email && !otp) {
        showMessage(elements.otpMessage, 'Please enter your email and OTP', 'error');
        return;
    } else if (!email) {
        showMessage(elements.otpMessage, 'Session expired. Please go back and request a new code.', 'error');
        return;
    } else if (!otp) {
        showMessage(elements.otpMessage, 'Please enter the verification code', 'error');
        return;
    }
    
    setLoading(elements.otpForm, true);
    
    try {
        const response = await fetch(`${API_BASE}/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            credentials: 'include',
            body: JSON.stringify({ email, otp })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            // Login successful
            const userResponse = await fetch(`${API_BASE}/me`, {
                credentials: 'include'
            });
            
            if (userResponse.ok) {
                const user = await userResponse.json();
                state.currentUser = user;
                state.pendingEmail = null;
                showDashboard(user);
            } else {
                showMessage(elements.otpMessage, 'Login successful but could not fetch user info', 'error');
            }
        } else {
            showMessage(elements.otpMessage, data.error || 'Invalid or expired OTP', 'error');
        }
    } catch (error) {
        console.error('OTP verification error:', error);
        showMessage(elements.otpMessage, 'Network error. Please try again.', 'error');
    } finally {
        setLoading(elements.otpForm, false);
    }
}

// Resend OTP
async function resendOTP() {
    const email = state.pendingEmail || elements.registerEmail?.value?.trim();
    
    if (!email) {
        showMessage(elements.otpMessage, 'No email address found. Please go back to registration.', 'error');
        return;
    }
    
    setLoading(elements.otpForm, true);
    
    try {
        const response = await fetch(`${API_BASE}/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            credentials: 'include',
            body: JSON.stringify({ email })
        });
        
        const data = await response.json();
        
        if (response.ok) {
            showMessage(elements.otpMessage, 'New verification code sent!', 'success');
        } else if (response.status === 429) {
            showMessage(elements.otpMessage, data.error || 'Too many requests. Please try again later.', 'error');
        } else {
            showMessage(elements.otpMessage, data.error || 'Failed to resend OTP', 'error');
        }
    } catch (error) {
        console.error('Resend OTP error:', error);
        showMessage(elements.otpMessage, 'Network error. Please try again.', 'error');
    } finally {
        setLoading(elements.otpForm, false);
    }
}

// Handle logout
async function handleLogout() {
    try {
        const response = await fetch(`${API_BASE}/logout`, {
            method: 'DELETE',
            credentials: 'include'
        });
        
        // Clear state regardless of response
        state.currentUser = null;
        state.pendingEmail = null;
        
        // Clear all OTP inputs
        if (elements.loginOtpInput) {
            elements.loginOtpInput.value = '';
        }
        if (elements.verifyOtpInput) {
            elements.verifyOtpInput.value = '';
        }
        
        showView('login');
    } catch (error) {
        console.error('Logout error:', error);
        // Still show login view on error
        state.currentUser = null;
        showView('login');
    }
}

// Set loading state on button
function setLoading(form, loading) {
    const btn = form?.querySelector('.btn-primary');
    if (btn) {
        btn.disabled = loading;
        if (loading) {
            btn.innerHTML = '<span class="loading"></span>Processing...';
        } else {
            // Reset button text based on form ID
            const formId = form?.id;
            if (formId === 'login-email-form') {
                btn.textContent = 'Send One-Time Password';
            } else if (formId === 'login-otp-form') {
                btn.textContent = 'Verify';
            } else if (formId === 'register-form') {
                btn.textContent = 'Send Code';
            } else if (formId === 'otp-form') {
                btn.textContent = 'Verify';
            }
        }
    }
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', init);
