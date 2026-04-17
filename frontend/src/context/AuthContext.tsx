import {
  createContext,
  useContext,
  type ParentComponent,
  createEffect,
} from "solid-js";
import { createStore } from "solid-js/store";
import type { User } from "../types";
import { authService } from "../services/authService";

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
}

interface AuthContextType extends AuthState {
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string) => Promise<void>;
  logout: () => void;
  refreshProfile: () => Promise<void>;
}

const defaultAuthContext: AuthContextType = {
  user: null,
  isAuthenticated: false,
  isLoading: true,
  login: async () => {},
  register: async () => {},
  logout: () => {},
  refreshProfile: async () => {},
};

const AuthContext = createContext<AuthContextType>(defaultAuthContext);

export const AuthProvider: ParentComponent = (props) => {
  const [state, setState] = createStore<AuthState>({
    user: null,
    isAuthenticated: false,
    isLoading: true,
  });

  // Load user profile on mount if token exists
  createEffect(async () => {
    if (authService.isAuthenticated()) {
      try {
        const user = await authService.getProfile();
        setState({ user, isAuthenticated: true, isLoading: false });
      } catch (error: any) {
        if (error?.status === 401 || error?.status === 403) {
          authService.logout();
          setState({ user: null, isAuthenticated: false, isLoading: false });
          return;
        }

        console.error("Failed to restore session:", error);
        setState({ isLoading: false });
      }
    } else {
      setState({ isLoading: false });
    }
  });

  const login = async (email: string, password: string) => {
    const response = await authService.login({ email, password });
    setState({ user: response.user, isAuthenticated: true });
  };

  const register = async (name: string, email: string, password: string) => {
    const response = await authService.register({ name, email, password });
    setState({ user: response.user, isAuthenticated: true });
  };

  const logout = () => {
    authService.logout();
    setState({ user: null, isAuthenticated: false });
  };

  const refreshProfile = async () => {
    if (authService.isAuthenticated()) {
      try {
        const user = await authService.getProfile();
        setState({ user });
      } catch (error) {
        console.error("Failed to refresh profile:", error);
      }
    }
  };

  const value: AuthContextType = {
    get user() {
      return state.user;
    },
    get isAuthenticated() {
      return state.isAuthenticated;
    },
    get isLoading() {
      return state.isLoading;
    },
    login,
    register,
    logout,
    refreshProfile,
  };

  return (
    <AuthContext.Provider value={value}>{props.children}</AuthContext.Provider>
  );
};

export const useAuth = () => {
  return useContext(AuthContext);
};
