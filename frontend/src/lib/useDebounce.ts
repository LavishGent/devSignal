import { useEffect, useState } from 'react'

// useDebounce delays a fast-changing value (like search text) so queries
// fire only after the user pauses typing.
export function useDebounce<T>(value: T, delayMs = 250): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), delayMs)
    return () => clearTimeout(t)
  }, [value, delayMs])
  return debounced
}
