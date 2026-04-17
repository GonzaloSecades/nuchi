export function chunkItems<T>(items: T[], _chunkSize: number) {
  const chunks: T[][] = [];

  for (let index = 0; index < items.length; index += _chunkSize) {
    chunks.push(items.slice(index, index + _chunkSize));
  }

  return chunks;
}
