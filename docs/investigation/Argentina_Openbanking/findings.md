# Findings — Argentina Open Banking / Sistema de Finanzas Abiertas

**Research date:** 2026-07-10 (web research; sources linked per item)

## 1. Background: no framework before 2025

For years Argentina had payments interoperability but no open banking regime.
The BCRA built instant payments (Transferencias 3.0, launched 2020–2022,
interoperable QR, CBU/CVU rails), while account **data** sharing remained a
private, commercial affair — e.g. MODO's bank-backed API access, a proxy
rather than an open ecosystem. Pre-2025 trackers described Argentina as
"contemplative": the BCRA was still weighing what primary legislation open
finance would need, with no compulsion on data holders.
Source: [Ozone Open Finance Tracker — Argentina](https://ozoneapi.com/the-open-finance-tracker/atlas/argentina/)
(note: that page predates the 2025 decree and is now stale; kept here as the
baseline picture).

## 2. The promise delivered: Decree 353/2025 creates the SFA

The Milei administration's financial-deregulation agenda produced
**Decree 353/2025**, published in the Boletín Oficial and effective
**2025-05-23**. It bundles three things: normative simplification for
investment, a simplified income-tax (Ganancias) filing regime, and the
creation of the **Sistema de Finanzas Abiertas (SFA)** — Argentina's first
open finance framework.

Key provisions:

- Natural and legal persons may, through **express consent**, share whatever
  financial information they consider pertinent with financial-system
  entities registered with the BCRA.
- Stated goals: credit development, competition, and financial inclusion.
- The **BCRA is the application authority**. It defines the parameters,
  standards, and requirements both for financial entities and for other
  national Executive-branch agencies that will participate (which is what
  opens the door to state-held data, e.g. ANSES and ARCA).

Sources:
[Boletín Oficial — Decreto 353/2025](https://www.boletinoficial.gob.ar/detalleAviso/primera/325767/20250523),
[Allende & Brea client note](https://allende.com/bancario/el-poder-ejecutivo-lanzo-el-open-finance-en-argentina-06-04-2025/),
[Palabras del Derecho](https://www.palabrasdelderecho.com.ar/articulo/5954/Crearon-el-Sistema-de-Finanzas-Abiertas),
[Blog del Contador](https://blogdelcontador.com.ar/news-45891-nuevo-regimen-simplificado-de-ganancias-y-sistema-de-finanzas-abiertas-claves-del-decreto-3532025).

## 3. Late 2025: BCRA turns it into an active program

Through H2 2025 the BCRA publicly framed the SFA as its lever to revive
credit:

- **2025-09** — Infobae details the BCRA plan: with user consent, a fintech
  can read someone's bank movements to offer better credit, or a bank can see
  wallet data to offer a mortgage.
  [Infobae, 2025-09-01](https://www.infobae.com/economia/2025/09/01/open-finance-en-la-argentina-como-es-el-plan-del-bcra-para-facilitar-el-acceso-al-credito/)
- **2025-10** — BCRA president **Santiago Bausili** announced the
  implementation push at a press conference, framing it as returning to each
  person control over their financial information.
  [Legal 500 Argentina fintech guide](https://www.legal500.com/guides/chapter/argentina-fintech/),
  [El Cronista](https://www.cronista.com/finanzas-mercados/bcra-aprobo-open-finance-de-que-se-trata-y-como-transformara-el-acceso-a-creditos/)
- **2025-11** — BCRA vice-president **Vladimir Werning** promoted the system
  at the Argentina Fintech Forum; La Nación coverage ("Prestar más y mejor").
  Industry datapoint from Mercado Pago: open-finance data carries ~20% weight
  in credit scoring and enables ~30% larger credit lines for previously
  unknown customers.
  [La Nación, 2025-11](https://www.lanacion.com.ar/economia/prestar-mas-y-mejor-como-funciona-el-open-finance-la-herramienta-que-impulsa-el-bcra-y-busca-revivir-nid12112025/),
  [LatamFintech](https://www.latamfintech.co/articles/como-funciona-el-open-finance-en-argentina-la-herramienta-del-bcra-que-busca-reactivar-el-credito)
- The model under discussion includes letting banks and fintechs access
  **ANSES and ARCA** data (income, employment, tax) alongside each other's
  account data, always behind user consent.
  [iProUP](https://www.iproup.com/finanzas/66980-banco-central-creditos-open-data-open-banking-finanzas-abiertas-plan),
  [Ámbito](https://www.ambito.com/finanzas/open-finance-bcra-bancos-y-fintechs-buscan-un-modelo-comun-ampliar-el-credito-n6186025)

## 4. 2026: infrastructure-definition phase

- **2025-12-29** — BCRA's *Objetivos y Planes 2026* commits to "continue to
  advance in the implementation of the Open Finance System, through the
  formation of technical groups for the delineation of the necessary
  infrastructure."
  [BCRA Objectives and Plans 2026](https://www.bcra.gob.ar/en/news/objectives-and-plans-2026/),
  [PDF (es)](https://www.bcra.gob.ar/archivos/Pdfs/Institucional/oyp-2026.pdf)
- **As of 2026-07-10** — no BCRA Comunicación "A" implementing SFA technical
  standards was found; the June 2026 communication indexes (through A 8445)
  contain nothing SFA-specific. The regime is voluntary, consent-based, and
  still without published API standards, an entity registry, or a formal
  rollout calendar.
- Sector estimates put an operational launch in **2026–2027**, drawing the
  comparison to Colombia's roughly two-year path from voluntary framework to
  mandatory regime.
  [JFC Attorneys fintech guide 2026](https://jfcattorneys.com/en/guides/fintech-regulation-argentina),
  [Latinia LATAM regulatory map 2026](https://latinia.com/en/resources/open-finance-in-latam-regulatory-map-2026)
- Supporting rails keep growing: the BCRA's April 2026 retail payments report
  counts ~99.6M interoperable QR transfer payments in pesos, 89 interoperable
  digital wallets, and 62 payment-with-transfer acceptors — the
  interoperability substrate the SFA would ride on.
  [Fiskil — Open Banking Argentina](https://www.fiskil.com/es/open-finance/argentina)

## 5. Assessment

| Question | Status as of 2026-07-10 |
| --- | --- |
| Legal basis | Yes — Decree 353/2025 (2025-05-23) |
| Regulator | BCRA (application authority) |
| Model | Voluntary, express-consent data sharing; credit-first framing |
| Implementing regulation | Not yet — technical working groups during 2026 |
| API standards | None published |
| Who can receive data | BCRA-registered financial-system entities; perimeter for non-financial apps undefined |
| State data (ANSES/ARCA) | Contemplated; parameters to be defined by BCRA |
| Expected launch | 2026–2027 (sector estimates, not official) |

The promise is real and moving, but as of mid-2026 there is nothing a
developer can integrate against yet. The binding milestone to watch is the
first BCRA Comunicación implementing the SFA.
