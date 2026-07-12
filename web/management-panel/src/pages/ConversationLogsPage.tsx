/**
 * ConversationLogsPage — PPAP-only viewer for the JSONL conversation log
 * store surfaced by the backend at /v0/management/conversation-logs.
 *
 * The page exposes:
 *   - cursor-paginated list of summary rows (timestamp, provider, model,
 *     status, latency, error indicator)
 *   - a simple filter row (provider, status_code, has_error)
 *   - a "Tail" mode that grabs the freshest N entries when polled or
 *     refreshed
 *   - a modal detail view that fetches the full entry (request +
 *     response bodies) and renders both as pretty-printed JSON.
 *
 * Keeps the page self-contained: no bulk edit, no destructive
 * operations. This is a read-only observability surface.
 */

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Modal } from '@/components/ui/Modal';
import { Select } from '@/components/ui/Select';
import { ToggleSwitch } from '@/components/ui/ToggleSwitch';
import { EmptyState } from '@/components/ui/EmptyState';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { useAuthStore, useNotificationStore } from '@/stores';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import {
  conversationLogsApi,
  type ConversationLogEntry,
  type ConversationLogListQuery,
  type ConversationLogSummary,
} from '@/services/api/conversationLogs';
import { getErrorMessage } from '@/utils/helpers';
import styles from './ConversationLogsPage.module.scss';

const DEFAULT_LIMIT = 50;

const formatTime = (iso: string, locale?: string) => {
  try {
    return new Intl.DateTimeFormat(locale, {
      dateStyle: 'short',
      timeStyle: 'medium',
    }).format(new Date(iso));
  } catch {
    return iso;
  }
};

const formatLatency = (ms?: number) => (ms == null ? '-' : `${Math.round(ms)} ms`);

const prettifyJSON = (value: unknown): string => {
  if (value === undefined || value === null || value === '') return '';
  if (typeof value === 'string') {
    // Attempt to prettify JSON-in-string; fall back to raw text.
    try {
      return JSON.stringify(JSON.parse(value), null, 2);
    } catch {
      return value;
    }
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
};

const buildEntryBodyPreview = (payload: ConversationLogEntry['request']): string => {
  if (!payload) return '';
  if (payload.body !== undefined && payload.body !== null && payload.body !== '') {
    return prettifyJSON(payload.body);
  }
  if (payload.text) return payload.text;
  if (payload.chunks && payload.chunks.length > 0) {
    return payload.chunks.join('');
  }
  return '';
};

interface DetailState {
  loading: boolean;
  entry: ConversationLogEntry | null;
  error: string | null;
}

const emptyDetail: DetailState = { loading: false, entry: null, error: null };

interface FilterState {
  provider: string;
  status_code: string;
  has_error: 'any' | 'yes' | 'no';
  tail: boolean;
}

const emptyFilter: FilterState = {
  provider: '',
  status_code: '',
  has_error: 'any',
  tail: false,
};

const buildQuery = (filter: FilterState, cursor?: string): ConversationLogListQuery => {
  const query: ConversationLogListQuery = { limit: DEFAULT_LIMIT };
  if (cursor) query.cursor = cursor;
  const provider = filter.provider.trim();
  if (provider) query.provider = provider;
  const statusRaw = filter.status_code.trim();
  if (statusRaw) {
    const parsed = Number(statusRaw);
    if (Number.isFinite(parsed) && parsed >= 0) query.status_code = parsed;
  }
  if (filter.has_error === 'yes') query.has_error = true;
  if (filter.has_error === 'no') query.has_error = false;
  return query;
};

export function ConversationLogsPage() {
  const { t, i18n } = useTranslation();
  const connectionStatus = useAuthStore((s) => s.connectionStatus);
  const { showNotification } = useNotificationStore();

  const [filter, setFilter] = useState<FilterState>(emptyFilter);
  const [entries, setEntries] = useState<ConversationLogSummary[]>([]);
  const [nextCursor, setNextCursor] = useState<string>('');
  const [enabled, setEnabled] = useState<boolean>(true);
  const [malformed, setMalformed] = useState<number>(0);
  const [loading, setLoading] = useState<boolean>(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<DetailState>(emptyDetail);

  const connected = connectionStatus === 'connected';

  const fetchPage = useCallback(
    async (options: { cursor?: string; append?: boolean } = {}) => {
      if (!connected) return;
      setLoading(true);
      try {
        const query = buildQuery(filter, options.cursor);
        const response = filter.tail
          ? await conversationLogsApi.tail(query)
          : await conversationLogsApi.list(query);
        setEnabled(response.enabled);
        setMalformed(response.malformed);
        setNextCursor(response.next_cursor ?? '');
        setEntries((prev) =>
          options.append ? [...prev, ...(response.entries ?? [])] : (response.entries ?? [])
        );
      } catch (err) {
        showNotification(
          `${t('ppap.conversationLogs.errors.load')}: ${getErrorMessage(err)}`,
          'error'
        );
      } finally {
        setLoading(false);
      }
    },
    [connected, filter, showNotification, t]
  );

  useEffect(() => {
    void fetchPage();
  }, [fetchPage]);

  useHeaderRefresh(() => fetchPage(), true);

  const openDetail = useCallback(
    async (id: string) => {
      setSelectedId(id);
      setDetail({ loading: true, entry: null, error: null });
      try {
        const response = await conversationLogsApi.get(id);
        setDetail({ loading: false, entry: response.entry, error: null });
      } catch (err) {
        const message = getErrorMessage(err);
        setDetail({ loading: false, entry: null, error: message });
        showNotification(`${t('ppap.conversationLogs.errors.detail')}: ${message}`, 'error');
      }
    },
    [showNotification, t]
  );

  const closeDetail = useCallback(() => {
    setSelectedId(null);
    setDetail(emptyDetail);
  }, []);

  const providers = useMemo(() => {
    const seen = new Set<string>();
    entries.forEach((row) => {
      if (row.provider) seen.add(row.provider);
    });
    return Array.from(seen).sort();
  }, [entries]);

  const requestPreview = detail.entry ? buildEntryBodyPreview(detail.entry.request) : '';
  const responsePreview = detail.entry ? buildEntryBodyPreview(detail.entry.response) : '';

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div>
          <h1 className={styles.title}>{t('ppap.conversationLogs.title')}</h1>
          <p className={styles.subtitle}>{t('ppap.conversationLogs.description')}</p>
        </div>
        <div className={styles.actions}>
          <label className={styles.tail}>
            <ToggleSwitch
              checked={filter.tail}
              onChange={(value) => setFilter((prev) => ({ ...prev, tail: value }))}
              ariaLabel={t('ppap.conversationLogs.tailToggle')}
            />
            <span>{t('ppap.conversationLogs.tailToggle')}</span>
          </label>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => void fetchPage()}
            disabled={loading || !connected}
          >
            {t('ppap.conversationLogs.refresh')}
          </Button>
        </div>
      </div>

      <div className={styles.filters}>
        <Input
          label={t('ppap.conversationLogs.filters.provider')}
          value={filter.provider}
          onChange={(event) => setFilter((prev) => ({ ...prev, provider: event.target.value }))}
          list="ppap-conversation-log-providers"
          placeholder={t('ppap.conversationLogs.filters.providerPlaceholder')}
        />
        <datalist id="ppap-conversation-log-providers">
          {providers.map((provider) => (
            <option key={provider} value={provider} />
          ))}
        </datalist>
        <Input
          label={t('ppap.conversationLogs.filters.statusCode')}
          value={filter.status_code}
          onChange={(event) => setFilter((prev) => ({ ...prev, status_code: event.target.value }))}
          placeholder="200"
          inputMode="numeric"
        />
        <div className="form-group">
          <label htmlFor="ppap-conversation-log-has-error">
            {t('ppap.conversationLogs.filters.hasError')}
          </label>
          <Select
            id="ppap-conversation-log-has-error"
            ariaLabel={t('ppap.conversationLogs.filters.hasError')}
            value={filter.has_error}
            onChange={(value) =>
              setFilter((prev) => ({ ...prev, has_error: value as FilterState['has_error'] }))
            }
            options={[
              { value: 'any', label: t('ppap.conversationLogs.filters.hasErrorAny') },
              { value: 'yes', label: t('ppap.conversationLogs.filters.hasErrorYes') },
              { value: 'no', label: t('ppap.conversationLogs.filters.hasErrorNo') },
            ]}
          />
        </div>
      </div>

      {!enabled ? (
        <EmptyState
          title={t('ppap.conversationLogs.disabledTitle')}
          description={t('ppap.conversationLogs.disabledHint')}
        />
      ) : entries.length === 0 && !loading ? (
        <EmptyState
          title={t('ppap.conversationLogs.emptyTitle')}
          description={t('ppap.conversationLogs.emptyHint')}
        />
      ) : (
        <div className={styles.tableWrapper}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>{t('ppap.conversationLogs.columns.timestamp')}</th>
                <th>{t('ppap.conversationLogs.columns.provider')}</th>
                <th>{t('ppap.conversationLogs.columns.model')}</th>
                <th>{t('ppap.conversationLogs.columns.status')}</th>
                <th>{t('ppap.conversationLogs.columns.path')}</th>
                <th>{t('ppap.conversationLogs.columns.hasError')}</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((row) => {
                const statusClass = row.has_error
                  ? styles.statusErr
                  : row.status_code && row.status_code >= 200 && row.status_code < 400
                    ? styles.statusOk
                    : '';
                return (
                  <tr
                    key={row.id}
                    className={`${styles.row} ${row.id === selectedId ? styles.rowSelected : ''}`}
                    onClick={() => void openDetail(row.id)}
                  >
                    <td>{formatTime(row.created_at, i18n.language)}</td>
                    <td>{row.provider || '-'}</td>
                    <td>{row.model || '-'}</td>
                    <td>
                      <span className={`${styles.statusPill} ${statusClass}`.trim()}>
                        {row.status_code ?? '-'}
                      </span>
                    </td>
                    <td>{row.path || '-'}</td>
                    <td>{row.has_error ? '⚠' : ''}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      <div className={styles.footer}>
        <span>
          {t('ppap.conversationLogs.counter', {
            count: entries.length,
            malformed,
          })}
        </span>
        <div className={styles.actions}>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => void fetchPage({ cursor: nextCursor, append: true })}
            disabled={!nextCursor || loading || filter.tail}
          >
            {t('ppap.conversationLogs.loadMore')}
          </Button>
        </div>
      </div>

      {selectedId ? (
        <Modal
          open
          onClose={closeDetail}
          title={t('ppap.conversationLogs.detail.title')}
          width={900}
        >
          <div className={styles.detailShell}>
            {detail.loading ? (
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: 24 }}>
                <LoadingSpinner />
                <span>{t('ppap.conversationLogs.detail.loading')}</span>
              </div>
            ) : detail.error ? (
              <EmptyState
                title={t('ppap.conversationLogs.detail.errorTitle')}
                description={detail.error}
              />
            ) : detail.entry ? (
              <>
                <dl className={styles.detailMeta}>
                  <div>
                    <dt>{t('ppap.conversationLogs.detail.fields.requestId')}</dt>
                    <dd>{detail.entry.request_id || '-'}</dd>
                  </div>
                  <div>
                    <dt>{t('ppap.conversationLogs.detail.fields.createdAt')}</dt>
                    <dd>{formatTime(detail.entry.created_at, i18n.language)}</dd>
                  </div>
                  <div>
                    <dt>{t('ppap.conversationLogs.detail.fields.completedAt')}</dt>
                    <dd>
                      {detail.entry.completed_at
                        ? formatTime(detail.entry.completed_at, i18n.language)
                        : '-'}
                    </dd>
                  </div>
                  <div>
                    <dt>{t('ppap.conversationLogs.detail.fields.latency')}</dt>
                    <dd>{formatLatency(detail.entry.latency_ms)}</dd>
                  </div>
                  <div>
                    <dt>{t('ppap.conversationLogs.detail.fields.provider')}</dt>
                    <dd>{detail.entry.provider || '-'}</dd>
                  </div>
                  <div>
                    <dt>{t('ppap.conversationLogs.detail.fields.model')}</dt>
                    <dd>{detail.entry.model || '-'}</dd>
                  </div>
                  <div>
                    <dt>{t('ppap.conversationLogs.detail.fields.method')}</dt>
                    <dd>{detail.entry.method || '-'}</dd>
                  </div>
                  <div>
                    <dt>{t('ppap.conversationLogs.detail.fields.path')}</dt>
                    <dd>{detail.entry.path || '-'}</dd>
                  </div>
                  <div>
                    <dt>{t('ppap.conversationLogs.detail.fields.upstream')}</dt>
                    <dd>{detail.entry.upstream_url || '-'}</dd>
                  </div>
                  <div>
                    <dt>{t('ppap.conversationLogs.detail.fields.status')}</dt>
                    <dd>{detail.entry.status_code ?? '-'}</dd>
                  </div>
                  {detail.entry.error ? (
                    <div>
                      <dt>{t('ppap.conversationLogs.detail.fields.error')}</dt>
                      <dd>{detail.entry.error}</dd>
                    </div>
                  ) : null}
                </dl>

                <div className={styles.codePane}>
                  <div className={styles.codePaneTitle}>
                    {t('ppap.conversationLogs.detail.request')}
                  </div>
                  <div className={styles.codePaneBody}>
                    <pre>{requestPreview || t('ppap.conversationLogs.detail.emptyBody')}</pre>
                  </div>
                </div>

                <div className={styles.codePane}>
                  <div className={styles.codePaneTitle}>
                    {t('ppap.conversationLogs.detail.response')}
                  </div>
                  <div className={styles.codePaneBody}>
                    <pre>{responsePreview || t('ppap.conversationLogs.detail.emptyBody')}</pre>
                  </div>
                </div>

                {detail.entry.usage !== undefined && detail.entry.usage !== null ? (
                  <div className={styles.codePane}>
                    <div className={styles.codePaneTitle}>
                      {t('ppap.conversationLogs.detail.usage')}
                    </div>
                    <div className={styles.codePaneBody}>
                      <pre>{prettifyJSON(detail.entry.usage)}</pre>
                    </div>
                  </div>
                ) : null}
              </>
            ) : null}
          </div>
        </Modal>
      ) : null}
    </div>
  );
}
