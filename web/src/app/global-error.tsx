"use client";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  console.error(
    JSON.stringify({ level: "ERROR", msg: "unhandled render error", error: error.message, digest: error.digest }),
  );

  return (
    <html>
      <body className="flex min-h-screen items-center justify-center p-8">
        <div className="text-center">
          <h2 className="text-lg font-semibold mb-2">Ocorreu um erro inesperado</h2>
          <button
            onClick={reset}
            className="mt-4 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            Tentar novamente
          </button>
        </div>
      </body>
    </html>
  );
}
