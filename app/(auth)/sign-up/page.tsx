'use client';

import Link from 'next/link';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Loader2, MailCheck } from 'lucide-react';
import { z } from 'zod';

import { AuthError } from '@/components/auth/auth-error';
import { AuthHeader } from '@/components/auth/auth-header';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { authErrorMessage, validationFieldErrors } from '@/lib/auth/errors';
import { useSession } from '@/lib/auth/session';

const formSchema = z.object({
  email: z.string().email('Enter a valid email address.'),
  // Mirrors the server's rule so the common mistake is caught without a
  // round-trip. The server re-checks by rune count, and its per-field response
  // is mapped onto this input if the two ever disagree.
  password: z.string().min(8, 'Password must be at least 8 characters.'),
});

type FormValues = z.infer<typeof formSchema>;

export default function SignUpPage() {
  const { register } = useSession();
  const [formError, setFormError] = useState<string | null>(null);
  const [sentMessage, setSentMessage] = useState<string | null>(null);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: { email: '', password: '' },
  });

  const onSubmit = async (values: FormValues) => {
    setFormError(null);
    try {
      const message = await register(values.email, values.password);
      setSentMessage(message);
    } catch (error) {
      // A field-level response belongs on the field that caused it, so the
      // user is not left scanning the form for what to fix.
      const fieldErrors = validationFieldErrors(error);
      for (const field of fieldErrors) {
        if (field.path === 'email' || field.path === 'password') {
          form.setError(field.path, { message: field.message });
        }
      }
      if (fieldErrors.length === 0) {
        setFormError(
          authErrorMessage(error, {
            EMAIL_ALREADY_REGISTERED:
              'That email is already registered. Try signing in instead.',
          })
        );
      }
    }
  };

  const disabled = form.formState.isSubmitting;

  // Registration issues no session on purpose — the account cannot sign in
  // until the emailed link is followed — so success is a distinct screen
  // rather than a redirect to the dashboard.
  if (sentMessage !== null) {
    return (
      <div className="space-y-6 text-center">
        <MailCheck className="mx-auto size-12 text-blue-500" />
        <AuthHeader title="Check your email" description={sentMessage} />
        <p className="text-sm text-[#7E8CA0]">
          The link expires, so if it stops working just sign up again with the
          same address.
        </p>
        <Button asChild variant="outline" className="w-full">
          <Link href="/sign-in">Back to sign in</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <AuthHeader
        title="Create an account"
        description="Start tracking your finances in a couple of minutes"
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
                    autoComplete="new-password"
                    disabled={disabled}
                    placeholder="••••••••"
                  />
                </FormControl>
                <FormDescription>At least 8 characters.</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <AuthError message={formError} />
          <Button className="w-full" type="submit" disabled={disabled}>
            {disabled && <Loader2 className="mr-2 size-4 animate-spin" />}
            Create account
          </Button>
        </form>
      </Form>
      <p className="text-center text-sm text-[#7E8CA0]">
        Already have an account?{' '}
        <Link href="/sign-in" className="text-blue-600 hover:underline">
          Sign in
        </Link>
      </p>
    </div>
  );
}
