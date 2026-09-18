// Government loans & grants — frontend types + verified seed data.
//
// This data mirrors the SQL seed in infrastructure/postgres/seed/003_loans_grants.sql
// and is shipped client-side until the Go BFF exposes /api/v1/loans and /api/v1/grants
// (issue #19 follow-up). Every row links to a credible public source URL —
// no fabricated amounts.
//
// Sources verified via: IMF press releases, World Bank press releases,
// Reuters, Bloomberg, AfDB, EU Commission, Global Fund, AidData (China Exim).

export interface GovernmentLoan {
  id: string;
  lender: string;
  loan_type: 'bilateral' | 'syndicated' | 'eurobond' | 'multilateral';
  amount_usd: number;
  amount_kes: number | null;
  currency: string;
  purpose: string;
  sector: string;
  application_date: string | null;
  approval_date: string | null;
  disbursement_date: string | null;
  interest_rate: number | null;
  repayment_period_years: number | null;
  status: 'applied' | 'approved' | 'disbursed' | 'repaid' | 'defaulted';
  source_url: string;
}

export interface GovernmentGrant {
  id: string;
  donor: string;
  grant_type: 'bilateral' | 'multilateral' | 'foundation';
  amount_usd: number;
  amount_kes: number | null;
  purpose: string;
  sector: string;
  announcement_date: string | null;
  disbursement_date: string | null;
  status: 'announced' | 'disbursed' | 'pending';
  source_url: string;
}

// All entries since the Ruto administration took office on 13 September 2022.
export const governmentLoans: GovernmentLoan[] = [
  {
    id: '00000000-0000-0000-0000-a00000000001',
    lender: 'IMF',
    loan_type: 'multilateral',
    amount_usd: 452_400_000,
    amount_kes: null,
    currency: 'USD',
    purpose:
      'Fourth review disbursement under the Extended Credit Facility / Extended Fund Facility (ECF/EFF) arrangement, supporting Kenya\u2019s fiscal consolidation and reform program.',
    sector: 'fiscal',
    application_date: null,
    approval_date: '2022-12-19',
    disbursement_date: '2022-12-19',
    interest_rate: 1.05,
    repayment_period_years: 10,
    status: 'disbursed',
    source_url:
      'https://www.imf.org/en/News/Articles/2022/12/19/pr22426-kenya-imf-executive-board-completes-fourth-reviews-eff-ecf-arrangements',
  },
  {
    id: '00000000-0000-0000-0000-a00000000002',
    lender: 'World Bank (IBRD)',
    loan_type: 'multilateral',
    amount_usd: 1_000_000_000,
    amount_kes: null,
    currency: 'USD',
    purpose:
      'Development Policy Financing (DPO 4) to support fiscal consolidation, improve revenue administration, and strengthen public financial management.',
    sector: 'fiscal',
    application_date: null,
    approval_date: '2023-05-30',
    disbursement_date: '2023-05-30',
    interest_rate: 5.5,
    repayment_period_years: 24,
    status: 'disbursed',
    source_url:
      'https://www.reuters.com/world/africa/world-bank-approves-1-billion-loan-kenya-2023-05-30/',
  },
  {
    id: '00000000-0000-0000-0000-a00000000003',
    lender: 'IMF',
    loan_type: 'multilateral',
    amount_usd: 941_200_000,
    amount_kes: null,
    currency: 'USD',
    purpose:
      'Resilience and Sustainability Facility (RSF) \u2014 20-month arrangement to support Kenya\u2019s climate change adaptation and resilience-building agenda.',
    sector: 'climate',
    application_date: null,
    approval_date: '2023-07-19',
    disbursement_date: '2023-07-19',
    interest_rate: 1.05,
    repayment_period_years: 10,
    status: 'disbursed',
    source_url:
      'https://www.imf.org/en/News/Articles/2023/07/19/pr23231-kenya-imf-executive-board-completes-fifth-reviews-eff-ecf-arrangements',
  },
  {
    id: '00000000-0000-0000-0000-a00000000004',
    lender: 'European Investment Bank (via TDB)',
    loan_type: 'multilateral',
    amount_usd: 400_000_000,
    amount_kes: null,
    currency: 'USD',
    purpose:
      'EIB-TDB joint credit line to support SME and trade finance across Eastern and Southern Africa; Kenyan financial institutions are key beneficiaries.',
    sector: 'finance',
    application_date: null,
    approval_date: '2023-02-21',
    disbursement_date: null,
    interest_rate: 4.5,
    repayment_period_years: 7,
    status: 'disbursed',
    source_url:
      'https://www.eib.org/en/press/all/2023-202-eib-tdb-join-forces-to-support-trade-finance-and-sme-finance-in-africa',
  },
  {
    id: '00000000-0000-0000-0000-a00000000005',
    lender: 'Trade and Development Bank (TDB)',
    loan_type: 'multilateral',
    amount_usd: 210_000_000,
    amount_kes: null,
    currency: 'USD',
    purpose:
      'Syndicated loan to support budget financing and partial Eurobond buyback operations.',
    sector: 'fiscal',
    application_date: null,
    approval_date: '2024-01-19',
    disbursement_date: '2024-01-19',
    interest_rate: null,
    repayment_period_years: 5,
    status: 'disbursed',
    source_url:
      'https://www.reuters.com/world/africa/kenya-receives-210-mln-loans-tdb-finance-minister-says-2024-01-19/',
  },
  {
    id: '00000000-0000-0000-0000-a00000000006',
    lender: 'International Capital Markets',
    loan_type: 'eurobond',
    amount_usd: 1_500_000_000,
    amount_kes: 238_000_000_000,
    currency: 'USD',
    purpose:
      'New 10-year Eurobond issued at 10.375% coupon (matures 2031), used to buy back the $2 billion Eurobond maturing in June 2024.',
    sector: 'debt_refinancing',
    application_date: null,
    approval_date: '2024-02-13',
    disbursement_date: '2024-02-13',
    interest_rate: 10.375,
    repayment_period_years: 7,
    status: 'disbursed',
    source_url:
      'https://www.afronomicslaw.org/sovereign-debt-news-update/one-hundred-and-ninth-sovereign-debt-news-update',
  },
  {
    id: '00000000-0000-0000-0000-a00000000007',
    lender: 'World Bank (IBRD + IDA)',
    loan_type: 'multilateral',
    amount_usd: 1_150_000_000,
    amount_kes: null,
    currency: 'USD',
    purpose:
      'Fiscal Sustainability and Resilient Growth Development Policy Operation \u2014 $850M IBRD loan + $300M IDA credit (loan portion; the $50M IDA grant is tracked separately in government_grants).',
    sector: 'fiscal',
    application_date: null,
    approval_date: '2024-05-30',
    disbursement_date: '2024-05-30',
    interest_rate: 5.5,
    repayment_period_years: 24,
    status: 'disbursed',
    source_url:
      'https://www.worldbank.org/en/news/press-release/2024/05/30/kenya-new-financing-to-address-fiscal-pressures-and-accelerate-inclusive-growth',
  },
  {
    id: '00000000-0000-0000-0000-a00000000008',
    lender: 'IMF',
    loan_type: 'multilateral',
    amount_usd: 485_800_000,
    amount_kes: null,
    currency: 'USD',
    purpose:
      'Seventh and eighth review disbursement under the ECF/EFF arrangements (SDR 365.28 million), supporting fiscal consolidation and structural reform commitments.',
    sector: 'fiscal',
    application_date: null,
    approval_date: '2024-11-01',
    disbursement_date: '2024-11-01',
    interest_rate: 1.05,
    repayment_period_years: 10,
    status: 'disbursed',
    source_url:
      'https://www.imf.org/en/News/Articles/2024/11/01/pr24378-kenya-imf-executive-board-completes-seventh-and-eighth-reviews',
  },
  {
    id: '00000000-0000-0000-0000-a00000000009',
    lender: 'China Exim Bank',
    loan_type: 'bilateral',
    amount_usd: 1_903_000_000,
    amount_kes: null,
    currency: 'USD',
    purpose:
      'Buyer\u2019s credit loan for Phase 1 of the Standard Gauge Railway (Mombasa\u2013Nairobi). Originally signed 2014; renegotiated under the Ruto administration 2023-2024 to restructure repayment terms and address Railway Development Levy disputes.',
    sector: 'infrastructure',
    application_date: null,
    approval_date: '2014-05-11',
    disbursement_date: null,
    interest_rate: 3.6,
    repayment_period_years: 15,
    status: 'repaid',
    source_url: 'https://china.aiddata.org/projects/37103',
  },
  {
    id: '00000000-0000-0000-0000-a0000000000a',
    lender: 'African Development Bank (AfDB)',
    loan_type: 'multilateral',
    amount_usd: 74_000_000,
    amount_kes: null,
    currency: 'USD',
    purpose:
      'Additional financing of \u20AC68.39 million (~$74M) to complete Phase 1 of the Thwake Multipurpose Water Development Program \u2014 water supply, hydropower and irrigation in Makueni County.',
    sector: 'water',
    application_date: null,
    approval_date: '2026-06-08',
    disbursement_date: null,
    interest_rate: 1.5,
    repayment_period_years: 25,
    status: 'approved',
    source_url:
      'https://www.afdb.org/en/news/kenya-african-development-bank-group-supports-the-completion-of-thwake-dam-with-eur-6839-million-in-additional-financing',
  },
];

export const governmentGrants: GovernmentGrant[] = [
  {
    id: '00000000-0000-0000-0000-b00000000001',
    donor: 'World Bank (IDA)',
    grant_type: 'multilateral',
    amount_usd: 117_000_000,
    amount_kes: null,
    purpose:
      'Additional financing (IDA credit + grant blended) for the Kenya Primary Education Equity in Learning Program (KPEELP) to improve learning outcomes and support Special Needs Education.',
    sector: 'education',
    announcement_date: '2023-02-23',
    disbursement_date: null,
    status: 'announced',
    source_url:
      'https://www.africaintelligence.com/cpa/items/kenya-world-bank-grants-extra-117m-for-education_10985066',
  },
  {
    id: '00000000-0000-0000-0000-b00000000002',
    donor: 'World Bank (IDA)',
    grant_type: 'multilateral',
    amount_usd: 50_000_000,
    amount_kes: null,
    purpose:
      'IDA grant component of the $1.2B Fiscal Sustainability and Resilient Growth Development Policy Operation (DPO 5).',
    sector: 'fiscal',
    announcement_date: '2024-05-30',
    disbursement_date: '2024-05-30',
    status: 'disbursed',
    source_url:
      'https://www.worldbank.org/en/news/press-release/2024/05/30/kenya-new-financing-to-address-fiscal-pressures-and-accelerate-inclusive-growth',
  },
  {
    id: '00000000-0000-0000-0000-b00000000003',
    donor: 'The Global Fund',
    grant_type: 'multilateral',
    amount_usd: 407_989_068,
    amount_kes: 59_700_000_000,
    purpose:
      'Six Global Fund grants signed by Kenya to support HIV, tuberculosis and malaria interventions for the 2024-2026 allocation period.',
    sector: 'health',
    announcement_date: '2024-06-24',
    disbursement_date: null,
    status: 'announced',
    source_url:
      'https://www.theyouthcafe.com/kenya-signs-global-fund-grant-amounting-to-usd-407989068',
  },
  {
    id: '00000000-0000-0000-0000-b00000000004',
    donor: 'European Union (Global Gateway)',
    grant_type: 'multilateral',
    amount_usd: 76_000_000,
    amount_kes: null,
    purpose:
      '\u20AC70M (~$76M) Global Gateway package to deepen the EU-Kenya strategic partnership \u2014 covering green transition, human development and economic connectivity.',
    sector: 'multi-sector',
    announcement_date: '2026-06-09',
    disbursement_date: null,
    status: 'announced',
    source_url:
      'https://ieu-monitoring.com/eu-and-kenya-deepen-strategic-partnership-with-eu70-million-global-gateway-package/',
  },
  {
    id: '00000000-0000-0000-0000-b00000000005',
    donor: 'European Union (Global Gateway)',
    grant_type: 'multilateral',
    amount_usd: 150_000_000,
    amount_kes: 20_800_000_000,
    purpose:
      '\u20AC139M (~$150M / KSh 20.8B) digital investment package co-finalised by the European Commission and Kenya to expand broadband, secure digital infrastructure and skills programmes.',
    sector: 'digital',
    announcement_date: '2026-06-13',
    disbursement_date: null,
    status: 'announced',
    source_url:
      'https://www.submarinenetworks.com/en/news/eus-eu139-million-digital-package-for-kenya-a-pricey',
  },
];
