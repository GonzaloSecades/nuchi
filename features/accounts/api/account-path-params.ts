export const accountPathParams = (id?: string) => {
  if (!id) {
    throw new Error('Account id is required');
  }

  return { path: { id } };
};
