import type { User } from '../types';

export function isAdmin(user: User | null | undefined): boolean {
  return !!user?.roles.includes('HR_ADMIN');
}

// Worker create / edit / deactivate / history / certifications — HR_ADMIN only.
// Heads of deps and sections cannot modify worker data.
export function canEditWorkers(user: User | null | undefined): boolean {
  return isAdmin(user);
}
