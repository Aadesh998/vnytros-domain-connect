import { createBrowserRouter, RouterProvider } from "react-router-dom"
import { AppLayout } from "@/components/layouts/AppLayout"
import { DomainsFeature } from "@/features/domains/DomainsFeature"
import { DashboardFeature } from "@/features/dashboard/DashboardFeature"
import { ApiKeysFeature } from "@/features/api-keys/ApiKeysFeature"
import { SettingsFeature } from "@/features/settings/SettingsFeature"
import { ProfileFeature } from "@/features/profile/ProfileFeature"
import { LoginPage } from "@/features/auth/LoginPage"
import { SignupPage } from "@/features/auth/SignupPage"
import { OAuthCallbackPage } from "@/features/auth/OAuthCallbackPage"
import {
  ForgotPasswordPage,
  ResetPasswordPage,
  VerifyEmailPage,
} from "@/features/auth/AccountRecoveryPages"
import { RequireAuth } from "@/features/auth/RequireAuth"
import { MailOverviewFeature } from "@/features/mail/MailOverviewFeature"
import { MailCampaignsFeature } from "@/features/mail/CampaignsFeature"
import { CampaignReportFeature } from "@/features/mail/CampaignReportFeature"
import { MailTemplatesFeature } from "@/features/mail/MailTemplatesFeature"
import { MailSettingsFeature } from "@/features/mail/MailSettingsFeature"

const router = createBrowserRouter([
  { path: "/login", element: <LoginPage /> },
  { path: "/signup", element: <SignupPage /> },
  { path: "/auth/:provider/callback", element: <OAuthCallbackPage /> },
  // The API's verification and reset emails link to FRONTEND_URL/verify and
  // FRONTEND_URL/reset-password; set FRONTEND_URL to this dashboard's URL.
  { path: "/verify", element: <VerifyEmailPage /> },
  { path: "/forgot-password", element: <ForgotPasswordPage /> },
  { path: "/reset-password", element: <ResetPasswordPage /> },
  {
    element: <RequireAuth />,
    children: [
      {
        path: "/",
        element: <AppLayout />,
        children: [
          { index: true, element: <DashboardFeature /> },
          { path: "domains", element: <DomainsFeature /> },
          { path: "developer", element: <ApiKeysFeature /> },
          { path: "settings", element: <SettingsFeature /> },
          { path: "profile", element: <ProfileFeature /> },
          { path: "mail", element: <MailOverviewFeature /> },
          { path: "mail/campaigns", element: <MailCampaignsFeature /> },
          { path: "mail/campaigns/:id", element: <CampaignReportFeature /> },
          { path: "mail/templates", element: <MailTemplatesFeature /> },
          { path: "mail/settings", element: <MailSettingsFeature /> },
        ],
      },
    ],
  },
])

export function App() {
  return <RouterProvider router={router} />
}
