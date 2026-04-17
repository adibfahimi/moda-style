import { createContext, useContext, type ParentComponent, createEffect } from 'solid-js';
import { createStore } from 'solid-js/store';
import type { CartItem, AddToCartRequest } from '../types';
import { cartService } from '../services/cartService';
import { useAuth } from './AuthContext';

interface CartState {
  items: CartItem[];
  subtotal: number;
  count: number;
  isLoading: boolean;
}

interface CartContextType extends CartState {
  addToCart: (data: AddToCartRequest) => Promise<void>;
  updateQuantity: (id: number, quantity: number) => Promise<void>;
  removeItem: (id: number) => Promise<void>;
  clearCart: () => Promise<void>;
  refreshCart: () => Promise<void>;
}

const defaultCartContext: CartContextType = {
  items: [],
  subtotal: 0,
  count: 0,
  isLoading: false,
  addToCart: async () => {},
  updateQuantity: async () => {},
  removeItem: async () => {},
  clearCart: async () => {},
  refreshCart: async () => {},
};

const CartContext = createContext<CartContextType>(defaultCartContext);

export const CartProvider: ParentComponent = (props) => {
  const auth = useAuth();
  const [state, setState] = createStore<CartState>({
    items: [],
    subtotal: 0,
    count: 0,
    isLoading: false,
  });

  const refreshCart = async () => {
    if (!auth.isAuthenticated) {
      setState({ items: [], subtotal: 0, count: 0 });
      return;
    }

    try {
      setState({ isLoading: true });
      const cart = await cartService.getCart();
      setState({ 
        items: cart.items, 
        subtotal: cart.subtotal, 
        count: cart.count,
        isLoading: false,
      });
    } catch (error) {
      console.error('Failed to fetch cart:', error);
      setState({ isLoading: false });
    }
  };

  createEffect(() => {
    if (auth.isAuthenticated && !auth.isLoading) {
      refreshCart();
    } else {
      setState({ items: [], subtotal: 0, count: 0 });
    }
  });

  const addToCart = async (data: AddToCartRequest) => {
    await cartService.addToCart(data);
    await refreshCart();
  };

  const updateQuantity = async (id: number, quantity: number) => {
    await cartService.updateCartItem(id, { quantity });
    await refreshCart();
  };

  const removeItem = async (id: number) => {
    await cartService.removeFromCart(id);
    await refreshCart();
  };

  const clearCart = async () => {
    await cartService.clearCart();
    await refreshCart();
  };

  const value: CartContextType = {
    get items() { return state.items; },
    get subtotal() { return state.subtotal; },
    get count() { return state.count; },
    get isLoading() { return state.isLoading; },
    addToCart,
    updateQuantity,
    removeItem,
    clearCart,
    refreshCart,
  };

  return (
    <CartContext.Provider value={value}>
      {props.children}
    </CartContext.Provider>
  );
};

export const useCart = () => {
  return useContext(CartContext);
};
