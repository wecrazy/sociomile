import { vi } from "vitest";

type MatchRule = string | RegExp | ((url: string, init?: RequestInit) => boolean);

export type MockRoute = {
  match: MatchRule;
  response: Response | ((url: string, init?: RequestInit) => Response | Promise<Response>);
};

const enLocale = `nav:
  dashboard: Dashboard
  conversations: Conversations
  settings: Settings
  tickets: Tickets
shell:
  eyebrow: Operations desk
  headline: Support orchestration
  workspaceLive: Live workspace
  workspaceChecking: Checking workspace
  workspaceOffline: Workspace offline
  backendService: Backend API
  workerService: Worker
  statusOnline: Online
  statusOffline: Offline
  statusChecking: Checking
  statusUnknown: Unknown
  tenant: Tenant
  operator: Operator
  demoCredentials: Demo credentials
auth:
  title: Sign in to your tenant workspace
  subtitle: Use the seeded demo credentials or your own account.
  demoHint: Pick a seeded role to prefill the form instantly.
  demoPasswordHint: "All seeded users share this password:"
  email: Email
  password: Password
  roleAdmin: Admin
  roleAgent: Agent
  submit: Sign In
  logout: Sign Out
  useDemoAccount: Use {{role}} login
  invalidCredentials: Invalid credentials.
dashboard:
  title: Operations overview
  subtitle: Monitor the current queue and move customers through the support lifecycle.
  openConversations: Conversations
  openTickets: Tickets
  totalWorkload: Open workload
  queueMix: Queue mix
settings:
  title: Workspace settings
  subtitle: Control theme and language preferences for the operator UI.
  profile: Profile
  preferences: Preferences
  language: Language
  darkMode: Switch to dark mode
  lightMode: Switch to light mode
conversation:
  title: Conversation queue
  customer: Customer
  channel: Channel
  status: Status
  assignedAgent: Assigned Agent
  assign: Assign conversation
  assignToAgent: Select an agent
  saveAssignment: Save Assignment
  reply: Reply
  sendReply: Send Reply
  viewTicket: View linked ticket
ticket:
  title: Ticket title
  titlePlural: Ticket queue
  description: Description
  status: Status
  priority: Priority
  assignedAgent: Assigned Agent
  escalate: Escalate to ticket
  create: Create Ticket
  updateStatus: Update ticket status
  saveStatus: Save Status
filters:
  allStatuses: All statuses
  allAgents: All agents
  allPriorities: All priorities
table:
  previous: Previous
  next: Next
  showing: Showing {{from}} to {{to}} of {{total}}
toast:
  signingIn: Signing in...
  loginSuccess: Signed in to Sociomile.
  loginFailed: We could not sign you in. Check your credentials and try again.
  logoutSuccess: Signed out of the workspace.
  languageChanged: Language switched to {{language}}.
  themeDark: Dark mode enabled.
  themeLight: Light mode enabled.
  assignmentSaving: Saving assignment...
  assignmentSaved: Conversation assignment saved.
  assignmentFailed: We could not save the assignment.
  replySending: Sending reply...
  replySent: Reply sent to the conversation.
  replyFailed: We could not send the reply.
  ticketCreating: Creating ticket...
  ticketCreated: Ticket created from the conversation.
  ticketCreateFailed: We could not create the ticket.
  ticketStatusSaving: Saving ticket status...
  ticketStatusSaved: Ticket status updated.
  ticketStatusFailed: We could not update the ticket status.
  conversationLoadFailed: We could not load that conversation.
  ticketLoadFailed: We could not load that ticket.
  viewTicketAction: View ticket
  updateAvailable: A new version is available.
  refreshAction: Refresh now
  workspaceRestored: Workspace connection restored.
common:
  loading: Loading...
  empty: No data available.
`;

const idLocale = `nav:
  dashboard: Dasbor
  conversations: Percakapan
  settings: Pengaturan
  tickets: Tiket
shell:
  eyebrow: Meja operasi
  headline: Orkestrasi dukungan
  workspaceLive: Workspace aktif
  workspaceChecking: Memeriksa workspace
  workspaceOffline: Workspace offline
  backendService: API backend
  workerService: Worker
  statusOnline: Online
  statusOffline: Offline
  statusChecking: Memeriksa
  statusUnknown: Tidak diketahui
  tenant: Tenant
  operator: Operator
  demoCredentials: Kredensial demo
auth:
  title: Masuk ke workspace tenant
  subtitle: Gunakan akun demo hasil seed atau akun Anda sendiri.
  demoHint: Pilih role hasil seed untuk langsung mengisi formulir.
  demoPasswordHint: "Semua user demo memakai kata sandi ini:"
  email: Email
  password: Kata sandi
  roleAdmin: Admin
  roleAgent: Agent
  submit: Masuk
  logout: Keluar
  useDemoAccount: Pakai login {{role}}
  invalidCredentials: Kredensial tidak valid.
dashboard:
  title: Ringkasan operasional
  subtitle: Pantau antrean saat ini dan gerakkan customer di sepanjang lifecycle support.
  openConversations: Percakapan
  openTickets: Tiket
  totalWorkload: Beban kerja terbuka
  queueMix: Komposisi antrean
settings:
  title: Pengaturan workspace
  subtitle: Atur preferensi tema dan bahasa untuk UI operator.
  profile: Profil
  preferences: Preferensi
  language: Bahasa
  darkMode: Ganti ke mode gelap
  lightMode: Ganti ke mode terang
conversation:
  title: Antrean percakapan
  customer: Customer
  channel: Channel
  status: Status
  assignedAgent: Agent
  assign: Assign percakapan
  assignToAgent: Pilih agent
  saveAssignment: Simpan assignment
  reply: Balas
  sendReply: Kirim Balasan
  viewTicket: Lihat tiket terkait
ticket:
  title: Judul tiket
  titlePlural: Antrean tiket
  description: Deskripsi
  status: Status
  priority: Prioritas
  assignedAgent: Agent
  escalate: Eskalasi menjadi tiket
  create: Buat Tiket
  updateStatus: Ubah status tiket
  saveStatus: Simpan Status
filters:
  allStatuses: Semua status
  allAgents: Semua agent
  allPriorities: Semua prioritas
table:
  previous: Sebelumnya
  next: Berikutnya
  showing: Menampilkan {{from}} sampai {{to}} dari {{total}}
toast:
  signingIn: Sedang masuk...
  loginSuccess: Berhasil masuk ke Sociomile.
  loginFailed: Kami tidak dapat memproses login. Periksa kredensial lalu coba lagi.
  logoutSuccess: Berhasil keluar dari workspace.
  languageChanged: Bahasa diubah ke {{language}}.
  themeDark: Mode gelap aktif.
  themeLight: Mode terang aktif.
  assignmentSaving: Menyimpan assignment...
  assignmentSaved: Assignment percakapan berhasil disimpan.
  assignmentFailed: Kami tidak dapat menyimpan assignment.
  replySending: Mengirim balasan...
  replySent: Balasan berhasil dikirim.
  replyFailed: Kami tidak dapat mengirim balasan.
  ticketCreating: Membuat tiket...
  ticketCreated: Tiket berhasil dibuat dari percakapan.
  ticketCreateFailed: Kami tidak dapat membuat tiket.
  ticketStatusSaving: Menyimpan status tiket...
  ticketStatusSaved: Status tiket berhasil diperbarui.
  ticketStatusFailed: Kami tidak dapat memperbarui status tiket.
  conversationLoadFailed: Kami tidak dapat memuat percakapan itu.
  ticketLoadFailed: Kami tidak dapat memuat tiket itu.
  viewTicketAction: Lihat tiket
  updateAvailable: Versi baru sudah tersedia.
  refreshAction: Muat ulang sekarang
  workspaceRestored: Koneksi workspace pulih.
common:
  loading: Memuat...
  empty: Tidak ada data.
`;

export function installFetchMock(routes: MockRoute[]) {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url =
      typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;

    for (const route of routes) {
      if (matches(route.match, url, init)) {
        if (route.response instanceof Response) {
          return route.response.clone();
        }

        return route.response(url, init);
      }
    }

    if (/\/health$/.test(url)) {
      return jsonResponse({
        status: "ok",
        port: 8080,
        services: {
          api: { status: "online" },
          worker: { status: "unknown" },
        },
      });
    }

    throw new Error(`Unhandled fetch: ${url}`);
  });

  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

export function localeRoutes(): MockRoute[] {
  return [
    { match: /\/locales\/en\.yaml$/, response: textResponse(enLocale) },
    { match: /\/locales\/id\.yaml$/, response: textResponse(idLocale) },
  ];
}

export function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: { "Content-Type": "application/json" },
  });
}

export function textResponse(body: string, init: ResponseInit = {}) {
  return new Response(body, {
    status: init.status ?? 200,
    headers: { "Content-Type": "text/plain" },
  });
}

function matches(rule: MatchRule, url: string, init?: RequestInit) {
  if (typeof rule === "string") {
    return url === rule;
  }

  if (rule instanceof RegExp) {
    return rule.test(url);
  }

  return rule(url, init);
}
