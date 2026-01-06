import { Loader2 } from 'lucide-react';
import { SignUp, ClerkLoaded, ClerkLoading } from '@clerk/nextjs';
import Image from 'next/image';

export default function Page() {
  return (
    <div className="grid min-h-screen grid-cols-1 lg:grid-cols-2">
      <div className="h-full flex-col items-center justify-center px-4 lg:flex">
        {/* <div className="space-y-4 pt-16 text-center">
          <h1 className="text-3xl font-bold text-[#2E2A47]">Welcome back</h1>
          <p className="text-base text-[#7E8CA0]">Please sign in to your account to continue</p>
        </div> */}
        <div className="mt-7 flex items-center justify-center">
          <ClerkLoaded>
            <SignUp path="/sign-up" />
          </ClerkLoaded>
          <ClerkLoading>
            <Loader2 className="text-muted-foreground animate-spin" />
          </ClerkLoading>
        </div>
      </div>
      <div className="hidden h-full items-center justify-center bg-blue-500 lg:flex">
        <Image src="/logo.svg" alt="Logo" width={200} height={200} />
      </div>
    </div>
  );
}
