'use client';

import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { Suspense, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { CircleCheck, CircleX, Loader2 } from 'lucide-react';
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
import { apiClient, unwrap } from '@/lib/api/client';
import { authErrorMessage, validationFieldErrors } from '@/lib/auth/errors';

const formSchema = z
  .object({
    password: z.string().min(8, 'Password must be at least 8 characters.'),
    confirmPassword: z.string().min(1, 'Confirm your new password.'),
  })
  // Client-side only, and deliberately so: the API takes a single password and
  // has no notion of confirmation. This exists to catch a typo in a field the
  // user cannot see, before it becomes a password they cannot reproduce.
  .refine((values) => values.password === values.confirmPassword, {
    path: ['confirmPassword'],
    message: 'Passwords do not match.',
  });

type FormValues = z.infer<typeof formSchema>;

/**
 * Sets a new password from the `?token=` a reset email carries.
 *
 * The path is fixed by the Go mailer (`resetPasswordPath = "/reset-password"`).
 * A successful reset revokes every outstanding session server-side, so this
 * page finishes by sending the user to sign in rather than establishing one.
 */
const ResetPassword = () => {
  const searchParams = useSearchParams();
  const token = searchParams.get('token');
  const [formError, setFormError] = useState<string | null>(null);
  const [done, setDone] = useState(false);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: { password: '', confirmPassword: '' },
  });

  const onSubmit = async (values: FormValues) => {
    setFormError(null);
    try {
      const result = await apiClient.POST('/auth/password-reset/confirm', {
        // Guarded by the missing-token screen below, so this is always a string
        // by the time the form can be submitted.
        body: { token: token as string, password: values.password },
      });
      unwrap(result, 'password reset');
      setDone(true);
    } catch (error) {
      for (const field of validationFieldErrors(error)) {
        if (field.path === 'password') {
          form.setError('password', { message: field.message });
        }
      }
      setFormError(
        authErrorMessage(error, {
          INVALID_TOKEN:
            'This reset link is invalid or has expired. Request a new one to continue.',
          // The token field is not something the user can act on, so a
          // validation failure naming it is reported as a link problem.
          VALIDATION_ERROR: 'Check the highlighted fields and try again.',
        })
      );
    }
  };

  const disabled = form.formState.isSubmitting;

  if (!token) {
    return (
      <div className="space-y-6 text-center">
        <CircleX className="mx-auto size-12 text-rose-500" />
        <AuthHeader
          title="Link incomplete"
          description="This reset link is missing its token. Open the link from your email directly, without editing it."
        />
        <Button asChild className="w-full">
          <Link href="/forgot-password">Request a new link</Link>
        </Button>
      </div>
    );
  }

  if (done) {
    return (
      <div className="space-y-6 text-center">
        <CircleCheck className="mx-auto size-12 text-emerald-500" />
        <AuthHeader
          title="Password updated"
          description="Your password has been reset and every other session was signed out."
        />
        <Button asChild className="w-full">
          <Link href="/sign-in">Continue to sign in</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <AuthHeader
        title="Choose a new password"
        description="Signing in again everywhere else will need the new one"
      />
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <FormField
            name="password"
            control={form.control}
            render={({ field }) => (
              <FormItem>
                <FormLabel>New password</FormLabel>
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
          <FormField
            name="confirmPassword"
            control={form.control}
            render={({ field }) => (
              <FormItem>
                <FormLabel>Confirm new password</FormLabel>
                <FormControl>
                  <Input
                    {...field}
                    type="password"
                    autoComplete="new-password"
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
            Reset password
          </Button>
        </form>
      </Form>
      <p className="text-center text-sm text-[#7E8CA0]">
        <Link href="/sign-in" className="text-blue-600 hover:underline">
          Back to sign in
        </Link>
      </p>
    </div>
  );
};

export default function ResetPasswordPage() {
  return (
    <Suspense
      fallback={<Loader2 className="text-muted-foreground mx-auto animate-spin" />}
    >
      <ResetPassword />
    </Suspense>
  );
}
