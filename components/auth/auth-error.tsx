import { AlertCircle } from 'lucide-react';

type Props = {
  message: string | null;
};

/**
 * The form-level error line shown above an auth form's submit button.
 *
 * `role="alert"` is what makes a failed sign-in reach a screen reader: the
 * message appears without any focus change, so without it the announcement
 * never happens and the user is left on a form that silently did nothing.
 */
export const AuthError = ({ message }: Props) => {
  if (!message) {
    return null;
  }

  return (
    <div
      role="alert"
      className="flex items-start gap-x-2 rounded-md bg-rose-50 p-3 text-sm text-rose-700"
    >
      <AlertCircle className="mt-0.5 size-4 shrink-0" />
      <span>{message}</span>
    </div>
  );
};
