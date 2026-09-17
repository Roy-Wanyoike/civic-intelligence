import { NextResponse } from 'next/server';

// This route proxies the Go API's refresh endpoint.
// Vercel Cron calls this hourly to trigger data refresh.
export async function POST() {
  const apiUrl = process.env.API_SERVICE_URL ?? 'http://localhost:9000';
  try {
    const resp = await fetch(`${apiUrl}/api/v1/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });
    const data = await resp.json();
    return NextResponse.json(data, { status: resp.status });
  } catch (error) {
    // If the Go API is not running, return a graceful response
    return NextResponse.json({
      status: 'skipped',
      message: 'Backend API not available — data refresh skipped. The Go API should be deployed separately.',
      refreshed_at: new Date().toISOString(),
    }, { status: 200 });
  }
}

// Also support GET for manual testing
export async function GET() {
  return POST();
}
