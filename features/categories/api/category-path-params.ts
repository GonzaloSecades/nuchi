export const categoryPathParams = (id?: string) => {
  if (!id) {
    throw new Error('Category id is required');
  }

  return { path: { id } };
};
