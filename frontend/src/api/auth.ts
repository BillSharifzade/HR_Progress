import { client } from './client';
import type { LoginResponse, User } from '../types';

export async function apiLogin(username: string, password: string): Promise<LoginResponse> {
  const r = await client.post<LoginResponse>('/auth/login', { username, password });
  return r.data;
}

export async function apiRefresh(): Promise<LoginResponse> {
  const r = await client.post<LoginResponse>('/auth/refresh');
  return r.data;
}

export async function apiLogout(): Promise<void> {
  await client.post('/auth/logout');
}

export async function apiMe(): Promise<User> {
  const r = await client.get<User>('/auth/me');
  return r.data;
}

