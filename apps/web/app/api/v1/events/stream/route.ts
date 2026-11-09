const apiURL = process.env.API_URL ?? "http://127.0.0.1:8080";

export async function GET(req: Request) {
  const cookie = req.headers.get("cookie") ?? "";
  const res = await fetch(`${apiURL}/api/v1/events/stream`, {
    headers: { cookie },
    cache: "no-store",
  });
  if (!res.body) {
    return new Response(await res.text(), {
      status: res.status,
      headers: { "Content-Type": res.headers.get("Content-Type") ?? "application/json" },
    });
  }
  return new Response(res.body, {
    status: res.status,
    headers: {
      "Content-Type": res.headers.get("Content-Type") ?? "text/event-stream",
      "Cache-Control": "no-cache, no-transform",
      Connection: "keep-alive",
      "X-Accel-Buffering": "no",
    },
  });
}
