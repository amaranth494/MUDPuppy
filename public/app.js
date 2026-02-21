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
    elements.loginForm = document.getElementById('login-form');
    elements.registerForm = document.getElementById('register-form');
    elements.otpForm = document.getElementById('otp-form');
    elements.loginEmail = document.getElementById('login-email');
    elements.registerEmail = document.getElementById('register-email');
    elements.otpInput = document.getElementById('otp-input') || document.getElementById('otp-input-verify');
    elements.userEmail = document.getElementById('user-email');
    elements.logoutBtn = document.getElementById('logout-btn');
    elements.loginMessage = document.getElementById('login-message');
    elements.registerMessage = document.getElementById('register-message');
    elements.otpMessage = document.getElementById('otp-message');
    
    // State for OTP inputs
    state.loginOtpValue = '';
    state.verifyOtpValue = '';
    
    // Cache OTP input in both sections
    elements.loginOtpInput = document.getElementById('otp-input');
    elements.verifyOtpInput = document.getElementById('otp-input-verify');
}

// Bind event listeners
function bindEvents() {
    // Login form
    elements.loginForm.addEventListener('submit', handleLoginSubmit);
    
    // Register form
    elements.registerForm.addEventListener('submit', handleRegisterSubmit);
    
    // OTP form
    elements.otpForm.addEventListener('submit', handleOTPSubmit);
    
    // Logout button
    elements.logoutBtn.addEventListener('click', handleLogout);
    
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

// Handle login form submission
async function handleLoginSubmit(e) {
    e.preventDefault();
    
    const email = elements.loginEmail?.value?.trim();
    const otp = elements.loginOtpInput?.value?.trim();
    
    if (!email || !otp) {
        showMessage(elements.loginMessage, 'Please enter email and OTP', 'error');
        return;
    }
    
    setLoading(elements.loginForm, true);
    
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
                showDashboard(user);
            } else {
                showMessage(elements.loginMessage, 'Login successful but could not fetch user info', 'error');
            }
        } else {
            showMessage(elements.loginMessage, data.error || 'Login failed', 'error');
        }
    } catch (error) {
        console.error('Login error:', error);
        showMessage(elements.loginMessage, 'Network error. Please try again.', 'error');
    } finally {
        setLoading(elements.loginForm, false);
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
            
            // Clear login email and use same for OTP
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
    
    if (!email || !otp) {
        showMessage(elements.otpMessage, 'Please enter your email and OTP', 'error');
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
            // Reset button text based on form
            if (form.id === 'login-form') {
                btn.textContent = 'Login';
            } else if (form.id === 'register-form') {
                btn.textContent = 'Send Code';
            } else if (form.id === 'otp-form') {
                btn.textContent = 'Verify';
            }
        }
    }
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', init);
