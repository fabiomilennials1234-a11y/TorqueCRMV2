/**
 * F08 Campanhas — CreateCampaignModal (S46).
 *
 * Wizard de 3 passos: (1) identidade, (2) audiência (filtro simples
 * DSL = filter de triggers S40), (3) agendamento + revisão. Submete via
 * useCreateCampaign e fecha. Navegação linear — "Voltar" volta um step,
 * "Próximo" avança mediante validação do passo atual.
 *
 * A DSL de audiência aqui reaproveita a forma do S40 `TriggerFilter`
 * ({all: [{field, op, value}]}) — o backend já aceita arbitrário unknown
 * e o worker de dispatch plugará no matcher em follow-up.
 */

import { useId, useMemo, useState } from 'react'

import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import { useCreateCampaign } from '@/hooks/useCampaigns'

type Step = 1 | 2 | 3

export function CreateCampaignModal({ onClose }: { onClose: () => void }) {
  const [step, setStep] = useState<Step>(1)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [template, setTemplate] = useState('')
  const [field, setField] = useState('origin')
  const [op, setOp] = useState<'eq' | 'in' | 'contains' | 'present'>('eq')
  const [value, setValue] = useState('')
  const [scheduledAt, setScheduledAt] = useState('')
  const create = useCreateCampaign()

  const nameId = useId()
  const descId = useId()
  const tplId = useId()
  const scheduleId = useId()

  const audienceQuery = useMemo(
    () =>
      op === 'present'
        ? { all: [{ field, op }] }
        : op === 'in'
          ? {
              all: [
                {
                  field,
                  op,
                  value: value
                    .split(',')
                    .map((v) => v.trim())
                    .filter(Boolean),
                },
              ],
            }
          : { all: [{ field, op, value: value.trim() }] },
    [field, op, value]
  )

  const canAdvance =
    (step === 1 && name.trim().length >= 2 && template.trim().length > 0) ||
    (step === 2 && field.trim().length > 0) ||
    step === 3

  async function handleSubmit() {
    const payload: Parameters<typeof create.mutateAsync>[0] = {
      name: name.trim(),
      template_body: template.trim(),
      audience_query: audienceQuery,
    }
    if (description.trim()) payload.description = description.trim()
    if (scheduledAt) payload.scheduled_at = new Date(scheduledAt).toISOString()
    await create.mutateAsync(payload)
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/60 p-4">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="campaign-modal-title"
        className="w-full max-w-lg rounded-xl bg-surface p-6 shadow-elev-2"
      >
        <div className="mb-4 flex items-center gap-2">
          <h2 id="campaign-modal-title" className="font-fraunces text-lg text-ink">
            Nova campanha
          </h2>
          <div className="ml-auto flex items-center gap-1.5">
            {[1, 2, 3].map((n) => (
              <span
                key={n}
                className={
                  'h-2 w-6 rounded-full ' +
                  (n <= step ? 'bg-accent' : 'bg-elevated/50')
                }
              />
            ))}
          </div>
        </div>

        {step === 1 && (
          <div className="space-y-3">
            <div>
              <label htmlFor={nameId} className="mb-1 block text-xs text-ink-muted">
                Nome
              </label>
              <Input
                id={nameId}
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Black Friday 2026"
              />
            </div>
            <div>
              <label htmlFor={descId} className="mb-1 block text-xs text-ink-muted">
                Descrição (opcional)
              </label>
              <Input
                id={descId}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>
            <div>
              <label htmlFor={tplId} className="mb-1 block text-xs text-ink-muted">
                Template da mensagem
              </label>
              <textarea
                id={tplId}
                value={template}
                onChange={(e) => setTemplate(e.target.value)}
                rows={5}
                placeholder="Olá {{nome}}, temos uma oferta..."
                className="w-full resize-y rounded-md bg-elevated/40 px-3 py-2 text-sm text-ink shadow-hairline placeholder:text-ink-dim focus:outline-none focus:ring-1 focus:ring-accent/50"
              />
            </div>
          </div>
        )}

        {step === 2 && (
          <div className="space-y-3">
            <p className="text-xs text-ink-dim">
              Defina um filtro sobre atributos do lead (DSL compartilhada com gatilhos de Copilot).
            </p>
            <div className="grid grid-cols-3 gap-2">
              <Input
                value={field}
                onChange={(e) => setField(e.target.value)}
                placeholder="Campo (origin, segment, ...)"
              />
              <select
                value={op}
                onChange={(e) => setOp(e.target.value as typeof op)}
                className="rounded-md bg-elevated/40 px-2 py-2 text-sm text-ink shadow-hairline focus:outline-none focus:ring-1 focus:ring-accent/50"
              >
                <option value="eq">eq</option>
                <option value="in">in (CSV)</option>
                <option value="contains">contains</option>
                <option value="present">present</option>
              </select>
              {op !== 'present' && (
                <Input
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                  placeholder={op === 'in' ? 'a, b, c' : 'valor'}
                />
              )}
            </div>
            <div className="rounded-md bg-elevated/30 p-3 text-2xs text-ink-dim">
              <span className="mr-2 font-semibold text-ink-muted">Preview:</span>
              <code className="font-mono">{JSON.stringify(audienceQuery)}</code>
            </div>
          </div>
        )}

        {step === 3 && (
          <div className="space-y-3">
            <div>
              <label htmlFor={scheduleId} className="mb-1 block text-xs text-ink-muted">
                Agendar (opcional)
              </label>
              <Input
                id={scheduleId}
                type="datetime-local"
                value={scheduledAt}
                onChange={(e) => setScheduledAt(e.target.value)}
              />
              <p className="mt-1 text-2xs text-ink-dim">
                Deixe vazio para ficar em draft — você pode lançar manualmente depois.
              </p>
            </div>
            <div className="rounded-md bg-elevated/30 p-3 text-xs">
              <div className="flex items-center gap-2">
                <Badge tone="neutral">Nome</Badge>
                <span className="truncate text-ink">{name}</span>
              </div>
              <div className="mt-1.5 flex items-center gap-2">
                <Badge tone="neutral">Template</Badge>
                <span className="truncate text-ink-dim">{template.slice(0, 60)}…</span>
              </div>
              <div className="mt-1.5 flex items-center gap-2">
                <Badge tone="neutral">Audiência</Badge>
                <code className="truncate font-mono text-2xs text-ink-dim">
                  {JSON.stringify(audienceQuery)}
                </code>
              </div>
            </div>
          </div>
        )}

        <div className="mt-5 flex items-center justify-between">
          <Button type="button" variant="ghost" size="sm" onClick={onClose}>
            Cancelar
          </Button>
          <div className="flex items-center gap-2">
            {step > 1 && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setStep((step - 1) as Step)}
              >
                Voltar
              </Button>
            )}
            {step < 3 ? (
              <Button
                type="button"
                variant="primary"
                size="sm"
                onClick={() => setStep((step + 1) as Step)}
                disabled={!canAdvance}
              >
                Próximo
              </Button>
            ) : (
              <Button
                type="button"
                variant="primary"
                size="sm"
                onClick={() => void handleSubmit()}
                disabled={create.isPending}
              >
                {create.isPending ? 'Criando…' : 'Criar campanha'}
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
