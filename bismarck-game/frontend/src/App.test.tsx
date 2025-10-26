import React from 'react';
import { render, screen } from '@testing-library/react';
import App from './App';

test('renders login form', () => {
  render(<App />);
  const loginTitle = screen.getByText(/Вход в игру/i);
  expect(loginTitle).toBeInTheDocument();
  
  const gameTitle = screen.getByText(/Погоня за Бисмарком/i);
  expect(gameTitle).toBeInTheDocument();
  
  const loginButton = screen.getByText(/Войти/i);
  expect(loginButton).toBeInTheDocument();
});
