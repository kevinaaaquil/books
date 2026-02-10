export async function register() {
  if (process.env.NEXT_RUNTIME === "nodejs") {
    const baseUrl = process.env.NEXT_PUBLIC_API_BASE_URL ?? "(not set)";
    console.log("[frontend] API base URL:", baseUrl);
  }
}
