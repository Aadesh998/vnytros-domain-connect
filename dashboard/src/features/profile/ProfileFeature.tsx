import { useState } from "react"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  AlertTriangle,
  Bell,
  Camera,
  Check,
  Globe2,
  KeyRound,
  Loader2,
  LogOut,
  Settings2,
  ShieldCheck,
  Trash2,
  UserCog,
  Webhook,
} from "lucide-react"
import { useUser } from "@/hooks/useUser"
import { clearTokens } from "@/lib/api/http"
import type { User } from "@/lib/user/types"
import { cn } from "@/lib/utils"

const FALLBACK_USER: User = {
  id: 1,
  email: "you@example.com",
  user_type: "user",
}

type SessionRow = {
  id: number
  device: string
  location: string
  ip: string
  lastActive: string
  current: boolean
}

const MOCK_SESSIONS: SessionRow[] = [
  {
    id: 1,
    device: "Chrome 138 · Windows 11",
    location: "Bengaluru, IN",
    ip: "103.21.244.18",
    lastActive: "Active now",
    current: true,
  },
  {
    id: 2,
    device: "Safari 17 · macOS Sonoma",
    location: "Bengaluru, IN",
    ip: "103.21.244.18",
    lastActive: "3 days ago",
    current: false,
  },
  {
    id: 3,
    device: "Vnytros iOS · iPhone 15",
    location: "Mumbai, IN",
    ip: "49.36.187.221",
    lastActive: "1 week ago",
    current: false,
  },
]

function deriveName(email: string): { first: string; last: string } {
  const local = email.split("@")[0] ?? ""
  const parts = local
    .split(/[._-]+/)
    .filter(Boolean)
    .map((p) => p.charAt(0).toUpperCase() + p.slice(1).toLowerCase())
  return { first: parts[0] ?? "", last: parts.slice(1).join(" ") ?? "" }
}

export function ProfileFeature() {
  const { user: realUser, isLoading } = useUser()
  const user = realUser ?? FALLBACK_USER
  const isFallback = !realUser && !isLoading

  if (isLoading && !realUser) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="size-6 animate-spin text-vn-accent" />
      </div>
    )
  }

  return (
    <div>
      <PageHeader />
      {isFallback && <DemoBanner />}

      <Tabs defaultValue="account" className="w-full">
        <TabsList className="mb-4">
          <TabsTrigger value="account" className="gap-2">
            <UserCog className="size-3.5" />
            Account
          </TabsTrigger>
          <TabsTrigger value="security" className="gap-2">
            <ShieldCheck className="size-3.5" />
            Security
          </TabsTrigger>
          <TabsTrigger value="notifications" className="gap-2">
            <Bell className="size-3.5" />
            Notifications
          </TabsTrigger>
          <TabsTrigger value="preferences" className="gap-2">
            <Settings2 className="size-3.5" />
            Preferences
          </TabsTrigger>
        </TabsList>

        <TabsContent value="account" className="space-y-4">
          <AccountTab user={user} />
        </TabsContent>

        <TabsContent value="security" className="space-y-4">
          <SecurityTab />
        </TabsContent>

        <TabsContent value="notifications" className="space-y-4">
          <NotificationsTab />
        </TabsContent>

        <TabsContent value="preferences" className="space-y-4">
          <PreferencesTab />
        </TabsContent>
      </Tabs>
    </div>
  )
}

function PageHeader() {
  return (
    <section className="mb-5">
      <span className="vn-eyebrow mb-2">profile</span>
      <h1 className="mt-2 text-3xl font-[540] tracking-[-0.035em] text-vn-text md:text-4xl">
        Account
      </h1>
      <p className="mt-2 max-w-2xl text-vn-text-2">
        Manage your personal information, security, notifications, and workspace preferences.
      </p>
    </section>
  )
}

function DemoBanner() {
  return (
    <div className="mb-4 flex items-start gap-3 rounded-lg border border-vn-warn/30 bg-vn-warn-soft p-3 text-[13px] text-vn-warn">
      <AlertTriangle className="size-4 shrink-0 mt-0.5" />
      <p>
        You're not signed in — showing example profile data. Sign in to see your real account.
      </p>
    </div>
  )
}

/* ─────────────────────────────────────────────────────────────
   ACCOUNT TAB
───────────────────────────────────────────────────────────── */

function AccountTab({ user }: { user: User }) {
  const derived = deriveName(user.email)
  const [firstName, setFirstName] = useState(derived.first || "Alex")
  const [lastName, setLastName] = useState(derived.last || "Morgan")
  const [phone, setPhone] = useState("+91 98765 43210")
  const [location, setLocation] = useState("Bengaluru, India")
  const [bio, setBio] = useState(
    "Building integrations on top of the Vnytros domain connect API.",
  )
  const [isDeleteOpen, setIsDeleteOpen] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState("")
  const roleLabel = user.user_type.charAt(0).toUpperCase() + user.user_type.slice(1)
  const initial = (firstName.charAt(0) || user.email.charAt(0)).toUpperCase()

  return (
    <>
      {/* Profile photo */}
      <Card className="overflow-hidden">
        <CardHeader className="border-b border-vn-hairline bg-vn-surface-2">
          <CardTitle className="text-sm">Profile photo</CardTitle>
          <CardDescription>
            This image appears in the sidebar and on shared resources.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex items-center gap-5 p-5">
          <div className="relative">
            <div className="flex size-20 items-center justify-center rounded-full border border-vn-accent/30 bg-vn-accent-soft text-3xl font-semibold text-vn-accent">
              {initial}
            </div>
            <div className="absolute -bottom-1 -right-1 flex size-7 items-center justify-center rounded-full border border-vn-hairline-2 bg-vn-surface text-vn-text-2">
              <Camera className="size-3.5" />
            </div>
          </div>
          <div className="flex flex-1 flex-col gap-2 sm:flex-row">
            <Button variant="outline" size="sm">
              <Camera className="size-3.5" />
              Upload new photo
            </Button>
            <Button variant="ghost" size="sm" className="text-vn-text-3">
              Remove
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Personal info */}
      <Card className="overflow-hidden">
        <CardHeader className="border-b border-vn-hairline bg-vn-surface-2">
          <CardTitle className="text-sm">Personal information</CardTitle>
          <CardDescription>
            How your name and contact details appear across Vnytros.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 p-5 sm:grid-cols-2">
          <Field label="First name" value={firstName} onChange={setFirstName} />
          <Field label="Last name" value={lastName} onChange={setLastName} />
          <Field
            label="Email"
            value={user.email}
            readOnly
            hint="Tied to your sign-in. Contact support to change."
          />
          <Field label="Phone" value={phone} onChange={setPhone} type="tel" />
          <Field
            label="Location"
            value={location}
            onChange={setLocation}
            className="sm:col-span-2"
          />
          <div className="grid gap-2 sm:col-span-2">
            <FieldLabel>About</FieldLabel>
            <Textarea
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              placeholder="A short bio shown on your profile."
            />
          </div>
        </CardContent>
        <CardFooter className="justify-end gap-2 border-t border-vn-hairline bg-vn-surface-2 px-5 py-3">
          <Button variant="outline" size="sm">Cancel</Button>
          <SaveButton onSave={() => {}} />
        </CardFooter>
      </Card>

      {/* Account meta */}
      <Card className="overflow-hidden">
        <CardHeader className="border-b border-vn-hairline bg-vn-surface-2">
          <CardTitle className="text-sm">Account</CardTitle>
          <CardDescription>Identifiers tied to your Vnytros account.</CardDescription>
        </CardHeader>
        <CardContent className="divide-y divide-vn-hairline p-0">
          <Row label="User ID" value={`#${user.id}`} mono />
          <Row
            label="Role"
            value={
              <Badge variant={user.user_type === "admin" ? "default" : "secondary"}>
                {roleLabel}
              </Badge>
            }
          />
          <Row label="Member since" value="January 12, 2026" />
        </CardContent>
      </Card>

      {/* Danger zone */}
      <Card className="overflow-hidden border-vn-danger/30 shadow-none">
        <CardHeader className="border-b border-vn-danger/20 bg-vn-danger-soft/40">
          <CardTitle className="text-sm text-vn-danger">Delete account</CardTitle>
          <CardDescription>
            Permanently delete your Vnytros account and all associated data. This cannot be undone.
          </CardDescription>
        </CardHeader>
        <CardFooter className="px-5 py-3">
          <Button variant="destructive" size="sm" onClick={() => setIsDeleteOpen(true)}>
            <Trash2 className="size-3.5" />
            Delete my account
          </Button>
        </CardFooter>
      </Card>

      <Dialog
        open={isDeleteOpen}
        onOpenChange={(open) => {
          if (!open) {
            setIsDeleteOpen(false)
            setDeleteConfirm("")
          }
        }}
      >
        <DialogContent className="sm:max-w-[440px]">
          <DialogHeader>
            <DialogTitle>Delete account</DialogTitle>
            <DialogDescription>
              This will permanently delete your account, API keys, domains, and all associated webhook logs. Type <span className="font-mono text-vn-text">DELETE</span> to confirm.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-2 py-2">
            <Input
              value={deleteConfirm}
              onChange={(e) => setDeleteConfirm(e.target.value)}
              placeholder="Type DELETE to confirm"
            />
          </div>
          <DialogFooter className="gap-2 sm:space-x-0">
            <Button variant="outline" onClick={() => setIsDeleteOpen(false)}>Cancel</Button>
            <Button
              variant="destructive"
              disabled={deleteConfirm !== "DELETE"}
              onClick={() => setIsDeleteOpen(false)}
            >
              Delete account
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

/* ─────────────────────────────────────────────────────────────
   SECURITY TAB
───────────────────────────────────────────────────────────── */

function SecurityTab() {
  const [current, setCurrent] = useState("")
  const [next, setNext] = useState("")
  const [confirm, setConfirm] = useState("")
  const [twoFactor, setTwoFactor] = useState(false)

  const handleSignOut = () => {
    clearTokens()
    window.location.href = "/"
  }

  return (
    <>
      {/* Password */}
      <Card className="overflow-hidden">
        <CardHeader className="border-b border-vn-hairline bg-vn-surface-2">
          <CardTitle className="text-sm">Password</CardTitle>
          <CardDescription>
            Use a strong, unique password to keep your account secure.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 p-5 sm:max-w-md">
          <Field
            label="Current password"
            value={current}
            onChange={setCurrent}
            type="password"
            placeholder="••••••••"
          />
          <Field
            label="New password"
            value={next}
            onChange={setNext}
            type="password"
            placeholder="At least 12 characters"
          />
          <Field
            label="Confirm new password"
            value={confirm}
            onChange={setConfirm}
            type="password"
            placeholder="Re-enter new password"
          />
        </CardContent>
        <CardFooter className="justify-end border-t border-vn-hairline bg-vn-surface-2 px-5 py-3">
          <SaveButton onSave={() => {}} label="Update password" />
        </CardFooter>
      </Card>

      {/* 2FA */}
      <Card className="overflow-hidden">
        <CardHeader className="border-b border-vn-hairline bg-vn-surface-2">
          <CardTitle className="text-sm">Two-factor authentication</CardTitle>
          <CardDescription>
            Add a second step to your sign-in using an authenticator app.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex items-center justify-between gap-4 p-5">
          <div className="flex items-start gap-3">
            <div className="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-md bg-vn-accent-soft text-vn-accent">
              <KeyRound className="size-4" />
            </div>
            <div>
              <p className="text-sm font-medium text-vn-text">
                {twoFactor ? "Authenticator app enabled" : "Not enabled"}
              </p>
              <p className="mt-0.5 text-xs text-vn-text-3">
                {twoFactor
                  ? "You'll be asked for a code from your app on every sign-in."
                  : "Recommended. Compatible with 1Password, Authy, Google Authenticator."}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <Switch
              checked={twoFactor}
              onChange={(e) => setTwoFactor(e.target.checked)}
              aria-label="Toggle two-factor authentication"
            />
          </div>
        </CardContent>
      </Card>

      {/* Sessions */}
      <Card className="overflow-hidden">
        <CardHeader className="border-b border-vn-hairline bg-vn-surface-2">
          <CardTitle className="text-sm">Active sessions</CardTitle>
          <CardDescription>
            Devices currently signed in to your account.
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Device</TableHead>
                <TableHead className="hidden md:table-cell">Location</TableHead>
                <TableHead>Last active</TableHead>
                <TableHead className="w-[120px] text-right" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {MOCK_SESSIONS.map((session) => (
                <TableRow key={session.id}>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-vn-text">{session.device}</span>
                      {session.current && (
                        <Badge variant="default" className="text-[10px] uppercase tracking-wider">
                          This device
                        </Badge>
                      )}
                    </div>
                    <p className="mt-0.5 font-mono text-[10px] text-vn-text-4">{session.ip}</p>
                  </TableCell>
                  <TableCell className="hidden text-vn-text-3 md:table-cell">
                    {session.location}
                  </TableCell>
                  <TableCell className="text-vn-text-3 text-xs">{session.lastActive}</TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={session.current ? handleSignOut : undefined}
                      className="text-vn-text-3 hover:text-vn-danger"
                    >
                      {session.current ? "Sign out" : "Revoke"}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
        <CardFooter className="justify-between border-t border-vn-hairline bg-vn-surface-2 px-5 py-3">
          <p className="text-xs text-vn-text-3">
            Don't recognize a device? Sign out everywhere and update your password.
          </p>
          <Button variant="outline" size="sm" onClick={handleSignOut}>
            <LogOut className="size-3.5" />
            Sign out everywhere
          </Button>
        </CardFooter>
      </Card>
    </>
  )
}

/* ─────────────────────────────────────────────────────────────
   NOTIFICATIONS TAB
───────────────────────────────────────────────────────────── */

function NotificationsTab() {
  const [emailPrefs, setEmailPrefs] = useState({
    accountActivity: true,
    securityAlerts: true,
    productUpdates: true,
    marketing: false,
    newsletter: false,
  })
  const [webhookPrefs, setWebhookPrefs] = useState({
    failures: true,
    weeklySummary: false,
    quotaWarnings: true,
  })

  const setEmail = (k: keyof typeof emailPrefs) => (v: boolean) =>
    setEmailPrefs((p) => ({ ...p, [k]: v }))
  const setWebhook = (k: keyof typeof webhookPrefs) => (v: boolean) =>
    setWebhookPrefs((p) => ({ ...p, [k]: v }))

  return (
    <>
      <Card className="overflow-hidden">
        <CardHeader className="border-b border-vn-hairline bg-vn-surface-2">
          <CardTitle className="text-sm">Email notifications</CardTitle>
          <CardDescription>
            Choose which emails you want to receive at your account address.
          </CardDescription>
        </CardHeader>
        <CardContent className="divide-y divide-vn-hairline p-0">
          <Toggle
            title="Account activity"
            description="Sign-ins from new devices, password changes, and email changes."
            checked={emailPrefs.accountActivity}
            onChange={setEmail("accountActivity")}
          />
          <Toggle
            title="Security alerts"
            description="Suspicious activity, leaked credential warnings, and 2FA changes."
            checked={emailPrefs.securityAlerts}
            onChange={setEmail("securityAlerts")}
          />
          <Toggle
            title="Product updates"
            description="New features, API changes, and platform announcements."
            checked={emailPrefs.productUpdates}
            onChange={setEmail("productUpdates")}
          />
          <Toggle
            title="Marketing & offers"
            description="Promotions, partner offers, and survey invitations."
            checked={emailPrefs.marketing}
            onChange={setEmail("marketing")}
          />
          <Toggle
            title="Monthly newsletter"
            description="Tips, best practices, and case studies — once a month, no more."
            checked={emailPrefs.newsletter}
            onChange={setEmail("newsletter")}
          />
        </CardContent>
        <CardFooter className="justify-end border-t border-vn-hairline bg-vn-surface-2 px-5 py-3">
          <SaveButton onSave={() => {}} label="Save preferences" />
        </CardFooter>
      </Card>

      <Card className="overflow-hidden">
        <CardHeader className="border-b border-vn-hairline bg-vn-surface-2">
          <CardTitle className="flex items-center gap-2 text-sm">
            <Webhook className="size-3.5 text-vn-text-3" /> Webhook & API alerts
          </CardTitle>
          <CardDescription>
            Operational alerts about your integration health.
          </CardDescription>
        </CardHeader>
        <CardContent className="divide-y divide-vn-hairline p-0">
          <Toggle
            title="Delivery failures"
            description="Email when a webhook endpoint fails 5 times in a row."
            checked={webhookPrefs.failures}
            onChange={setWebhook("failures")}
          />
          <Toggle
            title="Weekly delivery summary"
            description="Aggregate stats on webhook volume, success rate, and latency."
            checked={webhookPrefs.weeklySummary}
            onChange={setWebhook("weeklySummary")}
          />
          <Toggle
            title="API rate-limit warnings"
            description="Heads-up when your API keys start hitting rate limits."
            checked={webhookPrefs.quotaWarnings}
            onChange={setWebhook("quotaWarnings")}
          />
        </CardContent>
      </Card>
    </>
  )
}

/* ─────────────────────────────────────────────────────────────
   PREFERENCES TAB
───────────────────────────────────────────────────────────── */

function PreferencesTab() {
  const [theme, setTheme] = useState<"light" | "dark" | "system">("dark")
  const [language, setLanguage] = useState("en-US")
  const [timezone, setTimezone] = useState("Asia/Kolkata")
  const [dateFormat, setDateFormat] = useState("DD/MM/YYYY")
  const [density, setDensity] = useState<"comfortable" | "compact">("comfortable")

  return (
    <>
      <Card className="overflow-hidden">
        <CardHeader className="border-b border-vn-hairline bg-vn-surface-2">
          <CardTitle className="text-sm">Appearance</CardTitle>
          <CardDescription>
            Choose how Vnytros looks for you. Applies to this browser only.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 p-5 sm:grid-cols-3">
          <ThemeOption
            value="light"
            label="Light"
            preview="bg-[oklch(0.985_0.004_95)] border-vn-hairline-2"
            current={theme}
            onSelect={setTheme}
          />
          <ThemeOption
            value="dark"
            label="Dark"
            preview="bg-vn-surface border-vn-hairline-3"
            current={theme}
            onSelect={setTheme}
          />
          <ThemeOption
            value="system"
            label="System"
            preview="bg-gradient-to-br from-vn-surface to-[oklch(0.985_0.004_95)] border-vn-hairline-2"
            current={theme}
            onSelect={setTheme}
          />
        </CardContent>
        <CardContent className="border-t border-vn-hairline p-5">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-vn-text">Compact density</p>
              <p className="mt-0.5 text-xs text-vn-text-3">
                Tighter spacing for power users. Affects tables and lists.
              </p>
            </div>
            <Switch
              checked={density === "compact"}
              onChange={(e) => setDensity(e.target.checked ? "compact" : "comfortable")}
              aria-label="Toggle compact density"
            />
          </div>
        </CardContent>
      </Card>

      <Card className="overflow-hidden">
        <CardHeader className="border-b border-vn-hairline bg-vn-surface-2">
          <CardTitle className="flex items-center gap-2 text-sm">
            <Globe2 className="size-3.5 text-vn-text-3" /> Localization
          </CardTitle>
          <CardDescription>
            Language, timezone, and date format used across the dashboard.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 p-5 sm:grid-cols-2">
          <SelectField
            label="Language"
            value={language}
            onChange={setLanguage}
            options={[
              { value: "en-US", label: "English (United States)" },
              { value: "en-GB", label: "English (United Kingdom)" },
              { value: "es-ES", label: "Español" },
              { value: "fr-FR", label: "Français" },
              { value: "de-DE", label: "Deutsch" },
              { value: "ja-JP", label: "日本語" },
              { value: "hi-IN", label: "हिन्दी" },
            ]}
          />
          <SelectField
            label="Timezone"
            value={timezone}
            onChange={setTimezone}
            options={[
              { value: "Asia/Kolkata", label: "Asia/Kolkata · UTC+5:30" },
              { value: "UTC", label: "UTC" },
              { value: "America/New_York", label: "America/New York · UTC-5" },
              { value: "America/Los_Angeles", label: "America/Los Angeles · UTC-8" },
              { value: "Europe/London", label: "Europe/London · UTC+0" },
              { value: "Europe/Berlin", label: "Europe/Berlin · UTC+1" },
              { value: "Asia/Singapore", label: "Asia/Singapore · UTC+8" },
            ]}
          />
          <SelectField
            label="Date format"
            value={dateFormat}
            onChange={setDateFormat}
            options={[
              { value: "MM/DD/YYYY", label: "MM/DD/YYYY · 05/18/2026" },
              { value: "DD/MM/YYYY", label: "DD/MM/YYYY · 18/05/2026" },
              { value: "YYYY-MM-DD", label: "YYYY-MM-DD · 2026-05-18" },
            ]}
          />
        </CardContent>
        <CardFooter className="justify-end border-t border-vn-hairline bg-vn-surface-2 px-5 py-3">
          <SaveButton onSave={() => {}} label="Save preferences" />
        </CardFooter>
      </Card>
    </>
  )
}

/* ─────────────────────────────────────────────────────────────
   SHARED HELPERS
───────────────────────────────────────────────────────────── */

function Field({
  label,
  value,
  onChange,
  type = "text",
  placeholder,
  hint,
  readOnly,
  className,
}: {
  label: string
  value: string
  onChange?: (v: string) => void
  type?: string
  placeholder?: string
  hint?: string
  readOnly?: boolean
  className?: string
}) {
  return (
    <div className={cn("grid gap-2", className)}>
      <FieldLabel>{label}</FieldLabel>
      <Input
        type={type}
        value={value}
        placeholder={placeholder}
        readOnly={readOnly}
        onChange={onChange ? (e) => onChange(e.target.value) : undefined}
        className={readOnly ? "cursor-default opacity-80" : undefined}
      />
      {hint && <p className="text-[11px] text-vn-text-4">{hint}</p>}
    </div>
  )
}

function FieldLabel({ children }: { children: React.ReactNode }) {
  return (
    <label className="text-[11px] font-mono uppercase tracking-widest text-vn-text-3">
      {children}
    </label>
  )
}

function SelectField({
  label,
  value,
  onChange,
  options,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  options: { value: string; label: string }[]
}) {
  return (
    <div className="grid gap-2">
      <FieldLabel>{label}</FieldLabel>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="flex h-[42px] w-full rounded-md border border-vn-hairline-2 bg-vn-bg-2 px-3.5 py-2 text-sm text-vn-text shadow-sm transition-colors focus-visible:border-vn-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vn-accent/25"
      >
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
    </div>
  )
}

function Row({
  label,
  value,
  mono,
}: {
  label: string
  value: React.ReactNode
  mono?: boolean
}) {
  return (
    <div className="flex items-center justify-between gap-4 px-5 py-3.5">
      <p className="text-sm text-vn-text-3">{label}</p>
      <div className={cn("text-sm text-vn-text", mono && "font-mono text-xs")}>
        {value}
      </div>
    </div>
  )
}

function Toggle({
  title,
  description,
  checked,
  onChange,
}: {
  title: string
  description: string
  checked: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <div className="flex items-start justify-between gap-4 px-5 py-4">
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-vn-text">{title}</p>
        <p className="mt-0.5 text-xs text-vn-text-3">{description}</p>
      </div>
      <Switch
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        aria-label={title}
      />
    </div>
  )
}

function ThemeOption({
  value,
  label,
  preview,
  current,
  onSelect,
}: {
  value: "light" | "dark" | "system"
  label: string
  preview: string
  current: "light" | "dark" | "system"
  onSelect: (v: "light" | "dark" | "system") => void
}) {
  const isActive = current === value
  return (
    <button
      type="button"
      onClick={() => onSelect(value)}
      className={cn(
        "group flex flex-col gap-2 rounded-lg border p-3 text-left transition-colors",
        isActive
          ? "border-vn-accent/40 bg-vn-accent-soft"
          : "border-vn-hairline-2 hover:border-vn-hairline-3 hover:bg-vn-surface-2",
      )}
    >
      <div className={cn("h-16 w-full rounded-md border", preview)} />
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-vn-text">{label}</span>
        {isActive && <Check className="size-4 text-vn-accent" />}
      </div>
    </button>
  )
}

function SaveButton({
  onSave,
  label = "Save changes",
}: {
  onSave: () => void | Promise<void>
  label?: string
}) {
  const [state, setState] = useState<"idle" | "saving" | "saved">("idle")
  const handleClick = async () => {
    setState("saving")
    await Promise.resolve(onSave())
    await new Promise((r) => setTimeout(r, 500))
    setState("saved")
    setTimeout(() => setState("idle"), 1600)
  }
  return (
    <Button onClick={handleClick} disabled={state !== "idle"} size="sm">
      {state === "saving" && <Loader2 className="size-3.5 animate-spin" />}
      {state === "saved" && <Check className="size-3.5" />}
      {state === "idle" && label}
      {state === "saving" && "Saving…"}
      {state === "saved" && "Saved"}
    </Button>
  )
}
