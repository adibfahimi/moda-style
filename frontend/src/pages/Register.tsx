import { createSignal } from 'solid-js';
import { A, useNavigate } from '@solidjs/router';
import { useAuth } from '../context/AuthContext';
import Navbar from '../components/Navbar';

export default function Register() {
  const [name, setName] = createSignal('');
  const [email, setEmail] = createSignal('');
  const [password, setPassword] = createSignal('');
  const [confirmPassword, setConfirmPassword] = createSignal('');
  const [error, setError] = createSignal('');
  const [loading, setLoading] = createSignal(false);
  const [showPassword, setShowPassword] = createSignal(false);
  
  const auth = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setError('');

    if (password() !== confirmPassword()) {
      setError('Passwords do not match');
      return;
    }

    if (password().length < 6) {
      setError('Password must be at least 6 characters');
      return;
    }

    setLoading(true);

    try {
      await auth.register(name(), email(), password());
      navigate('/');
    } catch (err: any) {
      setError(err.message || 'Registration failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div class="min-h-screen bg-base-200 flex flex-col">
      <Navbar />
      
      <div class="flex-1 flex items-center justify-center p-4 py-12">
        <div class="card w-full max-w-md bg-base-100 shadow-2xl transition-all duration-300 hover:shadow-primary/10">
          <div class="card-body gap-6">
            {/* Header Section */}
            <div class="space-y-2 text-center">
              <h2 class="text-4xl font-black tracking-tight text-primary uppercase italic">
                Moda Style
              </h2>
              <p class="text-base-content/60 text-sm">Join our fashion community today.</p>
            </div>

            {/* Error Message */}
            {error() && (
              <div class="alert alert-error bg-error/10 border-error/20 text-error flex items-start gap-3 rounded-xl py-3">
                <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-5 w-5 mt-0.5" fill="none" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <span class="text-sm font-medium">{error()}</span>
              </div>
            )}

            <form onSubmit={handleSubmit} class="space-y-4">
              {/* Name Input */}
              <div class="form-control w-full">
                <label class="label pb-1">
                  <span class="label-text font-semibold uppercase tracking-wider text-xs opacity-70">Full Name</span>
                </label>
                <div class="relative">
                  <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-base-content/40">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                    </svg>
                  </div>
                  <input
                    type="text"
                    placeholder="John Doe"
                    class="input input-bordered w-full pl-10 focus:input-primary transition-all bg-base-200/50"
                    value={name()}
                    onInput={(e) => setName(e.currentTarget.value)}
                    required
                  />
                </div>
              </div>

              {/* Email Input */}
              <div class="form-control w-full">
                <label class="label pb-1">
                  <span class="label-text font-semibold uppercase tracking-wider text-xs opacity-70">Email Address</span>
                </label>
                <div class="relative">
                  <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-base-content/40">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 12a4 4 0 10-8 0 4 4 0 008 0zm0 0v1.5a2.5 2.5 0 005 0V12a9 9 0 10-9 9m4.5-1.206a8.959 8.959 0 01-4.5 1.206" />
                    </svg>
                  </div>
                  <input
                    type="email"
                    placeholder="name@company.com"
                    class="input input-bordered w-full pl-10 focus:input-primary transition-all bg-base-200/50"
                    value={email()}
                    onInput={(e) => setEmail(e.currentTarget.value)}
                    required
                  />
                </div>
              </div>

              {/* Password Input */}
              <div class="form-control w-full">
                <label class="label pb-1">
                  <span class="label-text font-semibold uppercase tracking-wider text-xs opacity-70">Password</span>
                </label>
                <div class="relative">
                  <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-base-content/40">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                    </svg>
                  </div>
                  <input
                    type={showPassword() ? "text" : "password"}
                    placeholder="Min. 6 characters"
                    class="input input-bordered w-full pl-10 pr-10 focus:input-primary transition-all bg-base-200/50"
                    value={password()}
                    onInput={(e) => setPassword(e.currentTarget.value)}
                    required
                  />
                  <button 
                    type="button"
                    onClick={() => setShowPassword(!showPassword())}
                    class="absolute inset-y-0 right-0 pr-3 flex items-center text-base-content/40 hover:text-primary transition-colors"
                  >
                    {showPassword() ? (
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l18 18" />
                      </svg>
                    ) : (
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                      </svg>
                    )}
                  </button>
                </div>
              </div>

              {/* Confirm Password */}
              <div class="form-control w-full">
                <label class="label pb-1">
                  <span class="label-text font-semibold uppercase tracking-wider text-xs opacity-70">Confirm Password</span>
                </label>
                <div class="relative">
                  <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-base-content/40">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                    </svg>
                  </div>
                  <input
                    type={showPassword() ? "text" : "password"}
                    placeholder="Repeat password"
                    class="input input-bordered w-full pl-10 focus:input-primary transition-all bg-base-200/50"
                    value={confirmPassword()}
                    onInput={(e) => setConfirmPassword(e.currentTarget.value)}
                    required
                  />
                </div>
              </div>

              {/* Submit Button */}
              <button
                type="submit"
                class="btn btn-primary w-full mt-2 normal-case text-lg shadow-lg shadow-primary/20"
                disabled={loading()}
              >
                {loading() ? (
                  <>
                    <span class="loading loading-spinner loading-sm"></span>
                    Creating Account...
                  </>
                ) : (
                  'Create Account'
                )}
              </button>
            </form>

      

            <p class="text-center text-sm text-base-content/60 mt-2">
              Already have an account?{' '}
              <A href="/login" class="link link-primary font-bold decoration-2 underline-offset-4">
                Sign in here
              </A>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}