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
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { apiClient, unwrap } from '@/lib/api/client';
import { authErrorMessage } from '@/lib/auth/errors';

const formSchema = z.object({
  email: z.string().email('Enter a valid email address.'),
});

type FormValues = z.infer<typeof formSchema>;

export default function ForgotPasswordPage() {
  const [sentMessage, setSentMessage] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: { email: '' },
  });

  const onSubmit = async (values: FormValues) => {
    setFormError(null);
    try {
      const result = await apiClient.POST('/auth/password-reset/request', {
        body: { email: values.email },
      });
      const { message } = unwrap(result, 'password reset request');
      setSentMessage(message);
    } catch (error) {
      setFormError(authErrorMessage(error));
    }
  };

  const disabled = form.formState.isSubmitting;

  // The API answers identically whether or not the address is registered, and
  // this screen must not undo that: it shows the server's own wording, with no
  // branch anywhere on whether an account was found. Anything that varied here
  // — different copy, a different icon, even a different delay — would turn
  // this form into an account-enumeration oracle.
  if (sentMessage !== null) {
    return (
      <div className="space-y-6 text-center">
        <MailCheck className="mx-auto size-12 text-blue-500" />
        <AuthHeader title="Check your email" description={sentMessage} />
        <Button asChild variant="outline" className="w-full">
          <Link href="/sign-in">Back to sign in</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <AuthHeader
        title="Reset your password"
        description="Enter your email and we'll send you a reset link"
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
          <AuthError message={formError} />
          <Button className="w-full" type="submit" disabled={disabled}>
            {disabled && <Loader2 className="mr-2 size-4 animate-spin" />}
            Send reset link
          </Button>
        </form>
      </Form>
      <p className="text-center text-sm text-[#7E8CA0]">
        Remembered it?{' '}
        <Link href="/sign-in" className="text-blue-600 hover:underline">
          Sign in
        </Link>
      </p>
    </div>
  );
}
