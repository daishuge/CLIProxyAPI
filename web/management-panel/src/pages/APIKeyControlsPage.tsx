/**
 * Structured management UI for `api-key-controls` — the per-downstream-key
 * whitelist + budget entries PPAP's fork extends the upstream config with.
 * Backend contract lives in internal/api/handlers/management/api_key_controls.go
 * and services/api/apiKeyControls.ts on the frontend side.
 *
 * The page shows every entry with its budget, live usage, USD spend, and
 * per-model breakdown. Operators can create a new key (server auto-generates
 * one if not supplied), toggle enabled, adjust `max-cost-usd`, or delete a key
 * without ever editing YAML.
 */

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { useAuthStore } from '@/stores';
import { apiKeyControlsApi } from '@/services/api';
import type { APIKeyControl, APIKeyControlRecentRequest } from '@/services/api';
import styles from './APIKeyControlsPage.module.scss';

function fmtUSD(n: number, digits = 4): string {
  if (!Number.isFinite(n)) return '—';
  if (n === 0) return '$0';
  if (Math.abs(n) < Math.pow(10, -digits)) return `< $${Math.pow(10, -digits).toFixed(digits)}`;
  return `$${n.toFixed(digits)}`;
}

function fmtInt(n: number): string {
  if (!Number.isFinite(n)) return '—';
  return n.toLocaleString();
}

function fmtDate(iso?: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}

interface CreateForm {
  name: string;
  apiKey: string;
  models: string;
  maxCostUSD: string;
  enabled: boolean;
}

const emptyCreateForm: CreateForm = {
  name: '',
  apiKey: '',
  models: 'claude-sonnet-*\nclaude-opus-*\nclaude-haiku-*',
  maxCostUSD: '20',
  enabled: true,
};

export function APIKeyControlsPage() {
  const { t } = useTranslation();
  const connectionStatus = useAuthStore((state) => state.connectionStatus);
  const disabled = connectionStatus !== 'connected';

  const [items, setItems] = useState<APIKeyControl[]>([]);
  const [externalPricingPath, setExternalPricingPath] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [creating, setCreating] = useState(false);
  const [createForm, setCreateForm] = useState<CreateForm>(emptyCreateForm);
  const [submitting, setSubmitting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await apiKeyControlsApi.list({ recent: 5 });
      setItems(data['api-key-controls'] ?? []);
      setExternalPricingPath(data.external_pricing_file || '');
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  useHeaderRefresh(load);
  useEffect(() => { void load(); }, [load]);

  const toggleExpand = (hash: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(hash)) next.delete(hash);
      else next.add(hash);
      return next;
    });
  };

  const toggleEnabled = async (item: APIKeyControl) => {
    if (disabled) return;
    const current = item.enabled === undefined || item.enabled === null ? true : item.enabled;
    try {
      await apiKeyControlsApi.patch({
        target_name: item.name,
        value: { enabled: !current },
      });
      await load();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setError(msg);
    }
  };

  const adjustBudget = async (item: APIKeyControl) => {
    if (disabled) return;
    const raw = window.prompt(
      t('api_key_controls.prompt_new_budget', { name: item.name }) ||
        `New max-cost-usd for ${item.name}?`,
      item['max-cost-usd'].toString(),
    );
    if (raw === null) return;
    const num = Number(raw);
    if (!Number.isFinite(num) || num < 0) {
      setError(t('api_key_controls.error_bad_budget') || 'invalid number');
      return;
    }
    try {
      await apiKeyControlsApi.patch({
        target_name: item.name,
        value: { 'max-cost-usd': num },
      });
      await load();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setError(msg);
    }
  };

  const deleteItem = async (item: APIKeyControl) => {
    if (disabled) return;
    if (!window.confirm(
      t('api_key_controls.confirm_delete', { name: item.name }) ||
        `Delete api-key-controls entry "${item.name}"?`,
    )) return;
    try {
      await apiKeyControlsApi.remove(item.name);
      await load();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setError(msg);
    }
  };

  const submitCreate = async () => {
    if (!createForm.name.trim()) {
      setError(t('api_key_controls.error_name_required') || 'name is required');
      return;
    }
    setSubmitting(true);
    setError('');
    try {
      const models = createForm.models
        .split('\n')
        .map((line) => line.trim())
        .filter(Boolean);
      const maxCost = Number(createForm.maxCostUSD);
      await apiKeyControlsApi.create({
        name: createForm.name.trim(),
        'api-key': createForm.apiKey.trim() || undefined,
        models: models.length ? models : undefined,
        'max-cost-usd': Number.isFinite(maxCost) && maxCost > 0 ? maxCost : undefined,
        enabled: createForm.enabled,
      });
      setCreating(false);
      setCreateForm(emptyCreateForm);
      await load();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      setError(msg);
    } finally {
      setSubmitting(false);
    }
  };

  const sortedItems = useMemo(() => items.slice().sort((a, b) => a.name.localeCompare(b.name)), [items]);

  return (
    <div className={styles.container}>
      <div className={styles.pageHeader}>
        <div className={styles.pageTitleGroup}>
          <h1 className={styles.pageTitle}>{t('api_key_controls.title') || 'API Key Controls'}</h1>
          <p className={styles.pageDescription}>
            {t('api_key_controls.description') ||
              'Per-downstream-key model whitelist, usage budget, and live consumption. Backed by PPAP api-key-controls.'}
          </p>
          {externalPricingPath ? (
            <p className={styles.pricingSource}>
              {t('api_key_controls.pricing_source') || 'Pricing source:'}{' '}
              <span className={styles.pricingSourcePath}>{externalPricingPath}</span>
            </p>
          ) : (
            <p className={styles.pricingSource}>
              {t('api_key_controls.pricing_builtin') ||
                'Pricing: built-in GPT table only (no model-pricing.json detected).'}
            </p>
          )}
        </div>
        <div className={styles.pageActions}>
          <button className={styles.secondaryButton} onClick={load} disabled={loading || disabled}>
            {loading ? (t('common.loading') || 'Loading...') : (t('common.refresh') || 'Refresh')}
          </button>
          <button
            className={styles.primaryButton}
            onClick={() => setCreating(true)}
            disabled={disabled}
          >
            {t('api_key_controls.create_button') || '+ New api-key'}
          </button>
        </div>
      </div>

      {error && <div className={styles.errorBox}>{error}</div>}

      {sortedItems.length === 0 && !loading ? (
        <div className={styles.emptyState}>
          {t('api_key_controls.empty') ||
            'No api-key-controls entries defined. Click "+ New api-key" to add one.'}
        </div>
      ) : (
        <div className={styles.list}>
          {sortedItems.map((item) => (
            <ItemCard
              key={item['api-key-hash'] || item.name}
              item={item}
              expanded={expanded.has(item['api-key-hash'] || item.name)}
              disabled={disabled}
              onToggle={() => toggleExpand(item['api-key-hash'] || item.name)}
              onToggleEnabled={() => toggleEnabled(item)}
              onAdjustBudget={() => adjustBudget(item)}
              onDelete={() => deleteItem(item)}
            />
          ))}
        </div>
      )}

      {creating && (
        <div className={styles.modal} onClick={() => setCreating(false)}>
          <div className={styles.modalCard} onClick={(e) => e.stopPropagation()}>
            <h2 className={styles.modalTitle}>
              {t('api_key_controls.create_title') || 'Create new api-key-controls entry'}
            </h2>
            <div className={styles.field}>
              <label>{t('api_key_controls.field_name') || 'Name'}</label>
              <input
                value={createForm.name}
                onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                placeholder="e.g. Bob-personal-20260718"
              />
            </div>
            <div className={styles.field}>
              <label>
                {t('api_key_controls.field_apikey') ||
                  'API key (leave blank to auto-generate as sk-ppap-<name>-<random>)'}
              </label>
              <input
                value={createForm.apiKey}
                onChange={(e) => setCreateForm({ ...createForm, apiKey: e.target.value })}
                placeholder="sk-ppap-…"
              />
            </div>
            <div className={styles.field}>
              <label>
                {t('api_key_controls.field_models') ||
                  'Model whitelist (one per line; empty = allow all)'}
              </label>
              <textarea
                value={createForm.models}
                onChange={(e) => setCreateForm({ ...createForm, models: e.target.value })}
              />
            </div>
            <div className={styles.field}>
              <label>
                {t('api_key_controls.field_budget') || 'max-cost-usd (0 = unlimited)'}
              </label>
              <input
                type="number"
                min="0"
                step="0.01"
                value={createForm.maxCostUSD}
                onChange={(e) => setCreateForm({ ...createForm, maxCostUSD: e.target.value })}
              />
            </div>
            <div className={styles.modalFooter}>
              <button className={styles.secondaryButton} onClick={() => setCreating(false)} disabled={submitting}>
                {t('common.cancel') || 'Cancel'}
              </button>
              <button className={styles.primaryButton} onClick={submitCreate} disabled={submitting}>
                {submitting ? (t('common.saving') || 'Saving...') : (t('common.create') || 'Create')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

interface ItemCardProps {
  item: APIKeyControl;
  expanded: boolean;
  disabled: boolean;
  onToggle: () => void;
  onToggleEnabled: () => void;
  onAdjustBudget: () => void;
  onDelete: () => void;
}

function ItemCard({ item, expanded, disabled, onToggle, onToggleEnabled, onAdjustBudget, onDelete }: ItemCardProps) {
  const { t } = useTranslation();

  const usage = item.usage;
  const budget = item['max-cost-usd'];
  const used = usage?.used_usd ?? 0;
  const percent = usage?.used_percent ?? (budget > 0 ? (used / budget) * 100 : 0);
  const enabled = item.enabled === undefined || item.enabled === null ? true : item.enabled;

  const modelBadges = (item.models ?? []).slice(0, 4).map((m) => (
    <code key={m} style={{ fontSize: 11, padding: '2px 6px', background: 'rgba(255,255,255,0.06)', borderRadius: 4, marginRight: 4 }}>
      {m}
    </code>
  ));

  return (
    <div className={styles.card}>
      <div className={styles.cardHeader}>
        <div className={styles.nameCol}>
          <div className={styles.name}>{item.name || '(unnamed)'}</div>
          <div>
            <span className={`${styles.badge} ${enabled ? styles.on : styles.off}`}>
              {enabled ? (t('common.enabled') || 'ENABLED') : (t('common.disabled') || 'DISABLED')}
            </span>
            {item.unlimited && (
              <span className={`${styles.badge} ${styles.warn}`}>
                {t('api_key_controls.unlimited') || 'UNLIMITED'}
              </span>
            )}
            {usage?.exhausted && (
              <span className={`${styles.badge} ${styles.danger}`}>
                {t('api_key_controls.exhausted') || 'BUDGET EXHAUSTED'}
              </span>
            )}
          </div>
          <div className={styles.subline}>
            <code>{item['api-key-mask']}</code> · sha256:{item['api-key-hash']}
          </div>
          <div style={{ marginTop: 4 }}>{modelBadges}{(item.models?.length ?? 0) > 4 && <span>+{(item.models!.length - 4)}</span>}</div>
        </div>

        <div className={styles.metric}>
          <div className={styles.metricLabel}>{t('api_key_controls.metric_requests') || 'Requests'}</div>
          <div className={styles.metricValue}>{fmtInt(usage?.total_requests ?? 0)}</div>
        </div>

        <div className={styles.progressWrap}>
          <div className={styles.metricLabel}>
            {budget > 0
              ? `${fmtUSD(used)} / ${fmtUSD(budget, 2)}`
              : t('api_key_controls.no_budget') || 'no budget cap'}
          </div>
          {budget > 0 && (
            <div className={styles.progressBar}>
              <div className={styles.progressFill} style={{ width: `${Math.min(100, Math.max(0, percent))}%` }} />
            </div>
          )}
        </div>

        <div className={styles.actions}>
          <button className={styles.secondaryButton} onClick={onToggle}>
            {expanded ? (t('api_key_controls.hide_details') || 'Hide') : (t('api_key_controls.show_details') || 'Details')}
          </button>
          <button className={styles.secondaryButton} onClick={onToggleEnabled} disabled={disabled}>
            {enabled ? (t('common.disable') || 'Disable') : (t('common.enable') || 'Enable')}
          </button>
          <button className={styles.secondaryButton} onClick={onAdjustBudget} disabled={disabled}>
            $
          </button>
          <button className={styles.dangerButton} onClick={onDelete} disabled={disabled}>
            {t('common.delete') || 'Delete'}
          </button>
        </div>
      </div>

      {expanded && usage && (
        <div className={styles.details}>
          <div className={styles.section}>
            <h4>{t('api_key_controls.by_model') || 'By model'}</h4>
            {usage.models.length === 0 ? (
              <div style={{ fontSize: 12, color: 'var(--color-text-secondary, #8a8f98)' }}>
                {t('api_key_controls.no_requests') || 'No priced requests yet.'}
              </div>
            ) : (
              <table className={styles.modelTable}>
                <thead>
                  <tr>
                    <th>{t('api_key_controls.model') || 'Model'}</th>
                    <th className="num">{t('api_key_controls.req') || 'Req'}</th>
                    <th className="num">{t('api_key_controls.tokens') || 'Tokens'}</th>
                    <th className="num">{t('api_key_controls.cost') || 'Cost'}</th>
                    <th>{t('api_key_controls.priced_by') || 'Priced by'}</th>
                  </tr>
                </thead>
                <tbody>
                  {usage.models.map((m) => (
                    <tr key={m.name}>
                      <td>{m.name}</td>
                      <td className="num">{fmtInt(m.requests)}</td>
                      <td className="num">{fmtInt(m.tokens)}</td>
                      <td className="num">{fmtUSD(m.used_usd)}</td>
                      <td>
                        <code style={{ fontSize: 11 }}>{m.price_source}</code>
                        {m.price_matched ? ` (${m.price_matched})` : ''}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>

          <div className={styles.section}>
            <h4>{t('api_key_controls.recent') || 'Recent requests'}</h4>
            {(usage.recent?.length ?? 0) === 0 ? (
              <div style={{ fontSize: 12, color: 'var(--color-text-secondary, #8a8f98)' }}>
                {t('api_key_controls.no_recent') || 'None yet.'}
              </div>
            ) : (
              <table className={styles.recentTable}>
                <thead>
                  <tr>
                    <th>{t('api_key_controls.time') || 'Time'}</th>
                    <th>{t('api_key_controls.model') || 'Model'}</th>
                    <th className="num">{t('api_key_controls.in') || 'In'}</th>
                    <th className="num">{t('api_key_controls.out') || 'Out'}</th>
                    <th className="num">{t('api_key_controls.ms') || 'ms'}</th>
                    <th className="num">{t('api_key_controls.cost') || 'Cost'}</th>
                  </tr>
                </thead>
                <tbody>
                  {(usage.recent ?? []).map((r: APIKeyControlRecentRequest, i: number) => (
                    <tr key={i}>
                      <td>{fmtDate(r.timestamp)}</td>
                      <td>{r.model}{r.failed && <span className={`${styles.badge} ${styles.danger}`} style={{ marginLeft: 6 }}>FAIL</span>}</td>
                      <td className="num">{fmtInt(r.input_tokens)}</td>
                      <td className="num">{fmtInt(r.output_tokens)}</td>
                      <td className="num">{fmtInt(r.latency_ms)}</td>
                      <td className="num">{fmtUSD(r.cost_usd, 6)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
