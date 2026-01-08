'use client';
import { usePathname } from 'next/navigation';
import { NavButton } from './nav-button';

const routes = [
  {
    href: '/',
    label: 'Overview',
  },
  {
    href: '/transactions',
    label: 'Transactions',
  },
  {
    href: '/accounts',
    label: 'Accounts',
  },
  {
    href: '/categories',
    label: 'Categories',
  },
  {
    href: '/settings',
    label: 'Settings',
  },
];

export const Navigation = () => {
  const pathname = usePathname();
  return (
    <nav className="hidden items-center gap-x-2 overflow-x-auto lg:flex">
      {routes.map((r) => (
        <NavButton key={r.href} href={r.href} label={r.label} isActive={pathname === r.href} />
      ))}
    </nav>
  );
};
