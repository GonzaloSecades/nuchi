import { SessionGuard } from '@/components/auth/session-guard';
import { Header } from '@/components/header';

type Props = {
  children: React.ReactNode;
};

const DashboardLayout = ({ children }: Props) => {
  return (
    <SessionGuard>
      <Header />
      <main className="px-3 lg:px-14">{children}</main>
    </SessionGuard>
  );
};

export default DashboardLayout;
