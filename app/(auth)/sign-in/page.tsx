'use client';

import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Loader2 } from 'lucide-react';
import { z } from 'zod';

import { AuthError } from '@/components/auth/auth-error';
import { AuthHeader } from '@/components/auth/auth-header';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { authErrorMessage } from '@/lib/auth/errors';
import { useSession } from '@/lib/auth/session';
import { restoredSessionRedirectTarget } from '@/lib/auth/redirect';

const formSchema = z.object({
  email: z.string().email('Enter a valid email address.'),
  // No minimum here. The server's length rule applies to new passwords, and
  // enforcing it on sign-in would reject an existing shorter password with a
  // client-side message instead of the honest "invalid email or password".
  password: z.string().min(1, 'Password is required.'),
});

type FormValues = z.infer<typeof formSchema>;

const SignInForm = () => {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { login, status } = useSession();
  const [formError, setFormError] = useState<string | null>(null);

  const restoredTarget = restoredSessionRedirectTarget(
    status,
    searchParams.get('redirect')
  );

  useEffect(() => {
    if (restoredTarget !== null) {
      router.replace(restoredTarget);
    }
  }, [restoredTarget, router]);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: { email: '', password: '' },
  });

  const onSubmit = async (values: FormValues) => {
    setFormError(null);
    try {
      await login(values.email, values.password);
    } catch (error) {
      setFormError(
        authErrorMessage(error, {
          EMAIL_NOT_VERIFIED:
            'Verify your email before signing in. Check your inbox for the link we sent.',
        })
      );
    }
  };

  const disabled = form.formState.isSubmitting;

  // A fresh document has only the httpOnly refresh cookie. Do not show a
  // sign-in form until bootstrap has decided whether that cookie restores a
  // session; an authenticated result is redirected by the effect above.
  if (status !== 'unauthenticated') {
    return (
      <div
        role="status"
        aria-label="Restoring session"
        className="flex min-h-40 items-center justify-center"
      >
        <Loader2 className="text-muted-foreground size-5 animate-spin" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <AuthHeader
        title="Welcome back"
        description="Please sign in to your account to continue"
      />
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <FormField
            name="email"
            control={form.control}
            render={({ field }) => (
              <FormItem>
                <FormLabel>Email</FormLabel>
                <FormControl>
                  <Input
                    {...field}
                    type="email"
                    autoComplete="email"
                    disabled={disabled}
                    placeholder="you@example.com"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            name="password"
            control={form.control}
            render={({ field }) => (
              <FormItem>
                <FormLabel>Password</FormLabel>
                <FormControl>
                  <Input
                    {...field}
                    type="password"
                    autoComplete="current-password"
                    disabled={disabled}
                    placeholder="••••••••"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <AuthError message={formError} />
          <Button className="w-full" type="submit" disabled={disabled}>
            {disabled && <Loader2 className="mr-2 size-4 animate-spin" />}
            Sign in
          </Button>
        </form>
      </Form>
      <div className="space-y-2 text-center text-sm text-[#7E8CA0]">
        <p>
          <Link
            href="/forgot-password"
            className="text-blue-600 hover:underline"
          >
            Forgot your password?
          </Link>
        </p>
        <p>
          Don&apos;t have an account?{' '}
          <Link href="/sign-up" className="text-blue-600 hover:underline">
            Sign up
          </Link>
        </p>
      </div>
    </div>
  );
};

export default function SignInPage() {
  // useSearchParams opts the page into client rendering; without a boundary
  // Next fails the production build rather than degrading at runtime.
  return (
    <Suspense
      fallback={
        <Loader2 className="text-muted-foreground mx-auto animate-spin" />
      }
    >
      <SignInForm />
    </Suspense>
  );
}
