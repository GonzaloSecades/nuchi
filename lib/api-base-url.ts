type ResolveApiBaseUrlOptions = {
  isBrowser?: boolean;
  publicApiUrl?: string;
};

export function resolveApiBaseUrl({
  isBrowser = typeof window !== 'undefined',
  publicApiUrl = process.env.NEXT_PUBLIC_API_URL,
}: ResolveApiBaseUrlOptions = {}) {
  if (isBrowser) {
    return '';
  }

  return publicApiUrl ?? '';
}
