import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function SettingsFeature() {
  return (
    <div>
      <section className="mb-5">
        <span className="vn-eyebrow mb-2">settings</span>
        <h1 className="mt-2 text-3xl font-[540] tracking-[-0.035em] text-vn-text md:text-4xl">
          Workspace settings
        </h1>
        <p className="mt-2 max-w-2xl text-vn-text-2">
          Manage workspace details and team access.
        </p>
      </section>

      <Tabs defaultValue="general" className="w-full">
        <TabsList>
          <TabsTrigger value="general">General</TabsTrigger>
          <TabsTrigger value="team">Team</TabsTrigger>
        </TabsList>

        <TabsContent value="general" className="mt-4 space-y-4">
          <Card className="overflow-hidden">
            <CardHeader className="border-b border-vn-hairline bg-vn-surface-2">
              <CardTitle className="text-sm">Workspace name</CardTitle>
              <CardDescription>
                This is your workspace's visible name within Vnytros.
              </CardDescription>
            </CardHeader>
            <CardContent className="p-6">
              <label className="mb-2 block font-mono text-[11px] uppercase tracking-widest text-vn-text-3">
                Name
              </label>
              <Input defaultValue="Personal Account" />
            </CardContent>
            <CardFooter className="border-t border-vn-hairline bg-vn-surface-2 px-6 py-4">
              <Button>Save changes</Button>
            </CardFooter>
          </Card>

          <Card className="overflow-hidden border-vn-danger/20 bg-vn-danger-soft shadow-none">
            <CardHeader className="border-b border-vn-danger/20">
              <CardTitle className="text-sm text-vn-danger">
                Delete workspace
              </CardTitle>
              <CardDescription>
                Permanently delete your workspace and all platform data. This
                action is not reversible.
              </CardDescription>
            </CardHeader>
            <CardFooter className="px-6 py-4">
              <Button variant="destructive">Delete workspace</Button>
            </CardFooter>
          </Card>
        </TabsContent>

        <TabsContent value="team" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Team members</CardTitle>
              <CardDescription>
                Invite your team to collaborate in this workspace.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-vn-text-3">
                Team management is not available yet.
              </p>
            </CardContent>
          </Card>
        </TabsContent>

      </Tabs>
    </div>
  )
}
