'use client';

import Link from 'next/link';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useSession } from '@/lib/auth/session';

export function Settings() {
  const { user } = useSession();

  return (
    <div className="mx-auto -mt-24 w-full max-w-(--breakpoint-2xl) pb-10">
      <Card className="border-none drop-shadow-sm">
        <CardHeader>
          <CardTitle className="text-xl">Settings</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="space-y-1">
            <h2 className="font-medium">Email address</h2>
            <p className="text-muted-foreground text-sm">{user?.email}</p>
          </div>
          <div className="space-y-3">
            <h2 className="font-medium">Password</h2>
            <p className="text-muted-foreground text-sm">
              Request an email with a link to reset your password.
            </p>
            <Button asChild variant="outline">
              <Link href="/forgot-password">Reset password</Link>
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
