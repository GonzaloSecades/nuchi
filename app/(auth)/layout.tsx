import Image from 'next/image';

/**
 * The split-screen shell every auth page renders into.
 *
 * Previously each page repeated this markup around a Clerk widget. With the
 * forms now app-owned there are five pages instead of two, so the frame lives
 * here once and the pages carry only their own content.
 */
export default function AuthLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <div className="grid min-h-screen grid-cols-1 lg:grid-cols-2">
      <div className="flex h-full flex-col items-center justify-center px-4 py-12">
        <div className="w-full max-w-md">{children}</div>
      </div>
      <div className="hidden h-full items-center justify-center bg-blue-500 lg:flex">
        <Image src="/logo.svg" alt="Logo" width={200} height={200} />
      </div>
    </div>
  );
}
