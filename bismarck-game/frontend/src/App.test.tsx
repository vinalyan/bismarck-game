import React from 'react';
import { render, screen } from '@testing-library/react';
import App from './App';

test('renders login form', () => {
  render(<App />);
  const loginElement = screen.getByText(/вход в игру/i);
  expect(loginElement).toBeInTheDocument();
});
