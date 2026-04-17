import { createSignal, Show, onMount } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import Navbar from '../components/Navbar';
import { useAuth } from '../context/AuthContext';
import { authService } from '../services/authService';

export default function Profile() {
  const auth = useAuth();
  const navigate = useNavigate();

  const [isEditing, setIsEditing] = createSignal(false);
  const [name, setName] = createSignal('');
  const [email, setEmail] = createSignal('');
  const [error, setError] = createSignal('');
  const [success, setSuccess] = createSignal('');
  const [loading, setLoading] = createSignal(false);

  onMount(() => {
    if (!auth.isAuthenticated) {
      navigate('/login');
      return;
    }

    if (auth.user) {
      setName(auth.user.name);
      setEmail(auth.user.email);
    }
  });

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    setLoading(true);

    try {
      await authService.updateProfile({
        name: name(),
        email: email(),
      });
      
      await auth.refreshProfile();
      setSuccess('Profile updated successfully!');
      setIsEditing(false);
    } catch (err: any) {
      setError(err.message || 'Failed to update profile');
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = () => {
    if (auth.user) {
      setName(auth.user.name);
      setEmail(auth.user.email);
    }
    setIsEditing(false);
    setError('');
    setSuccess('');
  };

  const handleLogout = () => {
    auth.logout();
    navigate('/login');
  };

  return (
    <div class="min-h-screen bg-base-200 flex flex-col">
      <Navbar />
      
      <div class="container mx-auto px-4 py-12 flex-1">
        <div class="max-w-3xl mx-auto space-y-8">
          {/* Header Section */}
          <div class="flex flex-col md:flex-row md:items-end justify-between gap-4 px-2">
            <div>
              <h2 class="text-xs uppercase tracking-widest font-bold text-primary mb-1">Account Settings</h2>
              <h1 class="text-4xl font-black text-base-content italic uppercase tracking-tighter">My Profile</h1>
            </div>
            <button
              onClick={handleLogout}
              class="btn btn-ghost btn-sm text-error normal-case gap-2"
            >
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
              </svg>
              Sign Out
            </button>
          </div>

          <Show when={auth.user}>
            <div class="card bg-base-100 shadow-2xl border border-base-300/50 overflow-hidden">
              {/* Profile Top Banner (Decorative) */}
              <div class="h-32 bg-linear-to-r from-primary/20 to-secondary/20" />

              <div class="card-body -mt-16 pt-0">
                {/* Avatar and Basic Info */}
                <div class="flex flex-col sm:flex-row items-center sm:items-end gap-6 mb-8 px-2">
                  <div class="avatar">
                    <div class="w-32 h-32 rounded-2xl ring ring-base-100 ring-offset-base-100 shadow-xl bg-base-200 flex items-center justify-center">
                      <span class="text-4xl font-black text-primary uppercase">
                        {auth.user?.name.charAt(0)}
                      </span>
                    </div>
                  </div>
                  <div class="text-center sm:text-left pb-2 flex-1">
                    <h3 class="text-2xl font-bold">{auth.user?.name}</h3>
                    <p class="text-base-content/60">{auth.user?.email}</p>
                  </div>
                </div>

                {/* Notifications */}
                <div class="px-2">
                  {error() && (
                    <div class="alert alert-error bg-error/10 border-error/20 text-error rounded-xl mb-6">
                      <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current h-5 w-5" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                      <span class="text-sm font-medium">{error()}</span>
                    </div>
                  )}

                  {success() && (
                    <div class="alert alert-success bg-success/10 border-success/20 text-success rounded-xl mb-6">
                      <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current h-5 w-5" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                      <span class="text-sm font-medium">{success()}</span>
                    </div>
                  )}
                </div>

                <Show
                  when={isEditing()}
                  fallback={
                    <div class="grid md:grid-cols-2 gap-8 px-2">
                      <div class="space-y-6">
                        <div class="group">
                          <label class="text-xs font-bold uppercase tracking-widest text-base-content/40 block mb-1">Full Name</label>
                          <div class="flex items-center gap-3 text-lg font-medium">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary/60" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                            </svg>
                            {auth.user?.name}
                          </div>
                        </div>

                        <div class="group">
                          <label class="text-xs font-bold uppercase tracking-widest text-base-content/40 block mb-1">Email Address</label>
                          <div class="flex items-center gap-3 text-lg font-medium">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary/60" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                            </svg>
                            {auth.user?.email}
                          </div>
                        </div>
                      </div>

                      <div class="space-y-6">
                         <div class="group">
                          <label class="text-xs font-bold uppercase tracking-widest text-base-content/40 block mb-1">Member Since</label>
                          <div class="flex items-center gap-3 text-lg font-medium">
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-primary/60" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                            </svg>
                            {new Date(auth.user!.created_at).toLocaleDateString('en-US', { month: 'long', year: 'numeric', day: 'numeric' })}
                          </div>
                        </div>
                        
                        <div class="bg-base-200/50 rounded-2xl p-4 border border-base-300">
                           <button
                            class="btn btn-primary btn-block normal-case"
                            onClick={() => setIsEditing(true)}
                          >
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                            </svg>
                            Edit Profile Details
                          </button>
                        </div>
                      </div>
                    </div>
                  }
                >
                  <form onSubmit={handleSubmit} class="space-y-6 px-2">
                    <div class="grid md:grid-cols-2 gap-6">
                      <div class="form-control w-full">
                        <label class="label pb-1">
                          <span class="label-text font-semibold uppercase tracking-wider text-xs opacity-70">Full Name</span>
                        </label>
                        <input
                          type="text"
                          class="input input-bordered w-full focus:input-primary bg-base-200/50"
                          value={name()}
                          onInput={(e) => setName(e.currentTarget.value)}
                          required
                        />
                      </div>

                      <div class="form-control w-full">
                        <label class="label pb-1">
                          <span class="label-text font-semibold uppercase tracking-wider text-xs opacity-70">Email Address</span>
                        </label>
                        <input
                          type="email"
                          class="input input-bordered w-full focus:input-primary bg-base-200/50"
                          value={email()}
                          onInput={(e) => setEmail(e.currentTarget.value)}
                          required
                        />
                      </div>
                    </div>

                    <div class="flex flex-col sm:flex-row gap-3 pt-4">
                      <button
                        type="submit"
                        class="btn btn-primary sm:flex-1 normal-case shadow-lg shadow-primary/20"
                        disabled={loading()}
                      >
                        {loading() ? (
                          <span class="loading loading-spinner loading-sm"></span>
                        ) : (
                          'Save Changes'
                        )}
                      </button>
                      <button
                        type="button"
                        class="btn btn-ghost sm:w-32 normal-case"
                        onClick={handleCancel}
                        disabled={loading()}
                      >
                        Cancel
                      </button>
                    </div>
                  </form>
                </Show>
              </div>
            </div>
          </Show>
        </div>
      </div>
    </div>
  );
}