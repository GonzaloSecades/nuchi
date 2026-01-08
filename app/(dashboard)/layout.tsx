import { Header } from '@/components/header';

type Props = {
  children: React.ReactNode;
};

const DashboardLayout = ({ children }: Props) => {
  return (
    <>
      <Header />
      <main className="lg:px14 px-3">{children}</main>
    </>
  );
};

export default DashboardLayout;
