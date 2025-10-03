const TokenKey = 'Admin-Token';

export function getToken() {
  return window.localStorage.getItem(TokenKey);
}

export function setToken(token) {
  window.localStorage.setItem(TokenKey, token);
  return token;
}

export function removeToken() {
  window.localStorage.removeItem(TokenKey);
}
