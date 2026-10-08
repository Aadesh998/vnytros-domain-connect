import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { FileText, Loader2, Pencil, Plus, Trash2 } from "lucide-react"
import { useMailTemplates } from "@/hooks/useMailTemplates"
import type { MailTemplate, TemplateStatus } from "@/lib/mail/types"
import { formatDateTime } from "./format"

export function MailTemplatesFeature() {
  const { templates, drafts, isLoading, error, create, update, remove } =
    useMailTemplates()

  const [editing, setEditing] = useState<MailTemplate | null>(null)
  const [isOpen, setIsOpen] = useState(false)
  const [name, setName] = useState("")
  const [subject, setSubject] = useState("")
  const [body, setBody] = useState("")
  const [status, setStatus] = useState<TemplateStatus>("draft")
  const [formError, setFormError] = useState("")
  const [isSaving, setIsSaving] = useState(false)

  const rows = [...templates, ...drafts]

  const openNew = () => {
    setEditing(null)
    setName("")
    setSubject("")
    setBody("")
    setStatus("draft")
    setFormError("")
    setIsOpen(true)
  }

  const openEdit = (template: MailTemplate) => {
    setEditing(template)
    setName(template.name)
    setSubject(template.subject)
    setBody(template.body)
    setStatus(template.status)
    setFormError("")
    setIsOpen(true)
  }

  const handleSave = async () => {
    if (!name.trim() || !subject.trim() || !body.trim()) {
      setFormError("Name, subject and body are all required")
      return
    }

    setIsSaving(true)
    setFormError("")
    try {
      const payload = {
        name: name.trim(),
        subject: subject.trim(),
        body,
        status,
      }
      if (editing) {
        await update(editing.id, payload)
      } else {
        await create(payload)
      }
      setIsOpen(false)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Could not save template")
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-vn-text">Templates</h1>
          <p className="mt-1 text-sm text-vn-text-3">
            Reusable HTML bodies. A campaign pins one template when it sends.
          </p>
        </div>
        <Button onClick={openNew}>
          <Plus className="size-4" />
          New template
        </Button>
      </div>

      {error && (
        <div className="rounded-xl border border-vn-danger/30 bg-vn-danger-soft p-4 text-sm text-vn-danger">
          {error}
        </div>
      )}

      {isLoading && rows.length === 0 ? (
        <div className="flex items-center gap-2 rounded-xl border border-vn-hairline bg-vn-surface p-6 text-sm text-vn-text-3">
          <Loader2 className="size-4 animate-spin" />
          Loading templates…
        </div>
      ) : rows.length === 0 ? (
        <div className="rounded-xl border border-vn-hairline bg-vn-surface p-8 text-center">
          <FileText className="mx-auto size-6 text-vn-text-4" />
          <p className="mt-3 text-sm font-medium text-vn-text">No templates yet</p>
          <p className="mt-1 text-sm text-vn-text-3">
            Create one to start building campaigns.
          </p>
        </div>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-vn-hairline bg-vn-surface">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Subject</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((template) => (
                <TableRow key={template.id}>
                  <TableCell className="font-medium text-vn-text">
                    {template.name}
                  </TableCell>
                  <TableCell className="max-w-xs truncate text-vn-text-2">
                    {template.subject}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        template.status === "published" ? "default" : "outline"
                      }
                    >
                      {template.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-vn-text-3">
                    {formatDateTime(template.created_at)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1.5">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => openEdit(template)}
                      >
                        <Pencil className="size-3.5" />
                        Edit
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label="Delete template"
                        onClick={() => void remove(template.id)}
                      >
                        <Trash2 className="size-3.5 text-vn-danger" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <Dialog open={isOpen} onOpenChange={setIsOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>
              {editing ? `Edit ${editing.name}` : "New template"}
            </DialogTitle>
            <DialogDescription>
              The body is sent as HTML. A watermark and tracking pixel are
              appended automatically at send time.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3">
            <div className="grid gap-3 sm:grid-cols-2">
              <div>
                <label className="mb-1.5 block text-sm text-vn-text-2">Name</label>
                <Input value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <div>
                <label className="mb-1.5 block text-sm text-vn-text-2">
                  Status
                </label>
                <select
                  value={status}
                  onChange={(e) => setStatus(e.target.value as TemplateStatus)}
                  className="w-full rounded-lg border border-vn-hairline-2 bg-vn-surface-2 px-3 py-2 text-sm text-vn-text focus:outline-none focus:ring-2 focus:ring-vn-accent/35"
                >
                  <option value="draft">Draft</option>
                  <option value="published">Published</option>
                </select>
              </div>
            </div>
            <div>
              <label className="mb-1.5 block text-sm text-vn-text-2">
                Subject line
              </label>
              <Input
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
              />
            </div>
            <div>
              <label className="mb-1.5 block text-sm text-vn-text-2">
                HTML body
              </label>
              <Textarea
                value={body}
                onChange={(e) => setBody(e.target.value)}
                rows={12}
                className="font-mono text-xs"
                placeholder="<h1>Hello</h1>"
              />
            </div>
            {formError && <p className="text-sm text-vn-danger">{formError}</p>}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setIsOpen(false)}>
              Cancel
            </Button>
            <Button onClick={() => void handleSave()} disabled={isSaving}>
              {isSaving && <Loader2 className="size-4 animate-spin" />}
              {editing ? "Save changes" : "Create template"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
