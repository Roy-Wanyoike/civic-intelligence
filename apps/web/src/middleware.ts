import { NextResponse, type NextRequest } from 'next/server';

// Country subdomain detection — ke.civicintelligence.com → country=KE
const COUNTRY_MAP: Record<string, string> = {
  ke: 'KE',
  ug: 'UG',
  tz: 'TZ',
  gh: 'GH',
  ng: 'NG',
  za: 'ZA',
};

export function middleware(request: NextRequest) {
  const host = request.headers.get('host') ?? '';
  const parts = host.split('.');
  
  // Check if the first part is a country code
  // e.g., ke.civicintelligence.com → KE
  if (parts.length >= 3) {
    const subdomain = parts[0].toLowerCase();
    const countryCode = COUNTRY_MAP[subdomain];
    
    if (countryCode) {
      // Add country header so API calls can use it
      const response = NextResponse.next();
      response.headers.set('x-civic-country', countryCode);
      
      // Also set a cookie for client-side access
      response.cookies.set('civic-country', countryCode, {
        path: '/',
        maxAge: 60 * 60 * 24 * 365, // 1 year
      });
      
      return response;
    }
  }
  
  // Default to Kenya for now
  return NextResponse.next();
}

export const config = {
  matcher: ['/((?!api|_next/static|_next/image|favicon.ico).*)'],
};
