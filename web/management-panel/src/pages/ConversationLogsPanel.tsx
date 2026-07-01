import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { EmptyState } from '@/components/ui/EmptyState';
import { Input } from '@/components/ui/Input';
import { Select } from '@/components/ui/Select';
import {
  IconCode,
  IconDownload,
  IconFileText,
  IconRefreshCw,
  IconSearch,
  IconX,
} from '@/components/ui/icons';
import {
  logsApi,
  type ConversationLogDetailResponse,
  type ConversationLogEntry,
  type ConversationLogPayload,
  type ConversationLogsQuery,
  type ConversationLogSummary,
} from '@/services/api/logs';
import { useAuthStore, useNotificationStore } from '@/stores';
import { copyToClipboard } from '@/utils/clipboard';
import { formatDateTime } from '@/utils/format';
import styles from './LogsPage.module.scss';

interface ConversationLogFilters {
  requestId: string;
  provider: string;
  model: string;
  path: string;
  statusCode: string;
  hasError: string;
  from: string;
  to: string;
  limit: string;
}

const DEFAULT_FILTERS: ConversationLogFilters = {
  requestId: '',
  provider: '',
  model: '',
  path: '',
  statusCode: '',
  hasError: '',
  from: '',
  to: '',
  limit: '50',
};

const payloadText = (payload?: ConversationLogPayload): string => {
  if (!payload) return '';
  if (payload.body !== undefined && payload.body !== null) {
    return JSON.stringify(payload.body, null, 2);
  }
  if (payload.text) return payload.text;
  if (Array.isArray(payload.chunks) && payload.chunks.length > 0) {
    return payload.chunks.join('\n');
  }
  return '';
};

const jsonText = (value: unknown): string => {
  if (value === undefined || value === null) return '';
  if (typeof value === 'string') return value;
  return JSON.stringify(value, null, 2);
};

const toRFC3339 = (value: string): string | undefined => {
  const trimmed = value.trim();
  if (!trimmed) return undefined;
  const parsed = new Date(trimmed);
  if (Number.isNaN(parsed.getTime())) return undefined;
  return parsed.toISOString();
};

const buildQuery = (filters: ConversationLogFilters, cursor?: string): ConversationLogsQuery => {
  const limit = Number.parseInt(filters.limit, 10);
  const statusCode = Number.parseInt(filters.statusCode, 10);
  const query: ConversationLogsQuery = {
    limit: Number.isFinite(limit) && limit > 0 ? limit : 50,
  };
  if (cursor) query.cursor = cursor;
  if (filters.requestId.trim()) query.request_id = filters.requestId.trim();
  if (filters.provider.trim()) query.provider = filters.provider.trim();
  if (filters.model.trim()) query.model = filters.model.trim();
  if (filters.path.trim()) query.path = filters.path.trim();
  if (Number.isFinite(statusCode) && statusCode >= 0) query.status_code = statusCode;
  if (filters.hasError === 'true') query.has_error = true;
  if (filters.hasError === 'false') query.has_error = false;
  const from = toRFC3339(filters.from);
  const to = toRFC3339(filters.to);
  if (from) query.from = from;
  if (to) query.to = to;
  return query;
};

const formatBytes = (bytes?: number): string => {
  const value = Number.isFinite(bytes) && bytes ? Math.max(bytes, 0) : 0;
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / 1024 / 1024).toFixed(2)} MB`;
};

function PayloadBlock({
  title,
  payload,
}: {
  title: string;
  payload?: ConversationLogPayload;
}) {
  const { t } = useTranslation();
  const text = payloadText(payload);
  const isEmpty = text.trim() === '';

  return (
    <div className={styles.conversationPayloadBlock}>
      <div className={styles.conversationSectionHeader}>
        <span>{title}</span>
        <div className={styles.conversationSectionMeta}>
          {payload?.truncated && (
            <span className={styles.conversationWarnBadge}>{t('logs.conversation_truncated')}</span>
          )}
          {typeof payload?.bytes === 'number' && <span>{formatBytes(payload.bytes)}</span>}
          {Array.isArray(payload?.chunks) && payload.chunks.length > 0 && (
            <span>{t('logs.conversation_chunks', { count: payload.chunks.length })}</span>
          )}
        </div>
      </div>
      {isEmpty ? (
        <div className={styles.conversationEmptyLine}>-</div>
      ) : (
        <pre className={styles.conversationCodeBlock} spellCheck={false}>
          {text}
        </pre>
      )}
    </div>
  );
}

function DetailSection({ title, value }: { title: string; value: unknown }) {
  const text = jsonText(value);
  if (!text) return null;

  return (
    <div className={styles.conversationPayloadBlock}>
      <div className={styles.conversationSectionHeader}>
        <span>{title}</span>
      </div>
      <pre className={styles.conversationCodeBlock} spellCheck={false}>
        {text}
      </pre>
    </div>
  );
}

function SummaryRow({
  entry,
  active,
  onSelect,
}: {
  entry: ConversationLogSummary;
  active: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      className={`${styles.conversationRow} ${active ? styles.conversationRowActive : ''}`}
      onClick={onSelect}
    >
      <div className={styles.conversationRowTop}>
        <span className={styles.conversationProvider}>{entry.provider || '-'}</span>
        <span className={entry.has_error ? styles.conversationErrorBadge : styles.conversationOkBadge}>
          {entry.status_code || (entry.has_error ? 'ERR' : '-')}
        </span>
      </div>
      <div className={styles.conversationRowTitle} title={entry.model || entry.id}>
        {entry.model || entry.id}
      </div>
      <div className={styles.conversationRowMeta}>
        <span title={entry.path}>{entry.path || '-'}</span>
        <span>{formatDateTime(entry.created_at)}</span>
      </div>
      {entry.request_id && (
        <div className={styles.conversationRequestId} title={entry.request_id}>
          {entry.request_id}
        </div>
      )}
    </button>
  );
}

function ConversationDetail({
  entry,
  summary,
  loading,
  error,
  onCopy,
}: {
  entry: ConversationLogEntry | null;
  summary?: ConversationLogSummary;
  loading: boolean;
  error: string;
  onCopy: () => void;
}) {
  const { t } = useTranslation();
  const hasError = Boolean(entry?.error) || Boolean(summary?.has_error);

  if (loading) {
    return <div className="hint">{t('common.loading')}</div>;
  }
  if (error) {
    return <div className="error-box">{error}</div>;
  }
  if (!entry) {
    return (
      <EmptyState
        title={t('logs.conversation_detail_empty_title')}
        description={t('logs.conversation_detail_empty_desc')}
      />
    );
  }

  return (
    <div className={styles.conversationDetailBody}>
      <div className={styles.conversationDetailHeader}>
        <div className={styles.conversationDetailTitle}>
          <span>{entry.model || entry.id}</span>
          <span
            className={hasError ? styles.conversationErrorBadge : styles.conversationOkBadge}
          >
            {entry.status_code || (hasError ? 'ERR' : '-')}
          </span>
        </div>
        <Button variant="secondary" size="sm" onClick={onCopy}>
          <span className={styles.buttonContent}>
            <IconCode size={15} />
            {t('logs.conversation_copy_json')}
          </span>
        </Button>
      </div>

      <div className={styles.conversationMetaGrid}>
        <div>
          <span>{t('logs.conversation_provider')}</span>
          <strong>{entry.provider || '-'}</strong>
        </div>
        <div>
          <span>{t('logs.conversation_path')}</span>
          <strong title={entry.path}>{entry.path || '-'}</strong>
        </div>
        <div>
          <span>{t('logs.conversation_started')}</span>
          <strong>{formatDateTime(entry.created_at)}</strong>
        </div>
        <div>
          <span>{t('logs.conversation_latency')}</span>
          <strong>{entry.latency_ms ? `${entry.latency_ms} ms` : '-'}</strong>
        </div>
        <div>
          <span>{t('logs.conversation_request_id')}</span>
          <strong title={entry.request_id}>{entry.request_id || '-'}</strong>
        </div>
        <div>
          <span>{t('logs.conversation_line_size')}</span>
          <strong>{formatBytes(summary?.line_bytes)}</strong>
        </div>
      </div>

      {entry.error && <div className="error-box">{entry.error}</div>}

      <PayloadBlock title={t('logs.conversation_request_payload')} payload={entry.request} />
      <PayloadBlock title={t('logs.conversation_response_payload')} payload={entry.response} />
      <DetailSection title={t('logs.conversation_usage')} value={entry.usage} />
      <DetailSection title={t('logs.conversation_metadata')} value={entry.metadata} />
      <DetailSection title={t('logs.conversation_request_headers')} value={entry.request_headers} />
      <DetailSection title={t('logs.conversation_response_headers')} value={entry.response_headers} />
      {entry.upstream_url && (
        <div className={styles.conversationPayloadBlock}>
          <div className={styles.conversationSectionHeader}>
            <span>{t('logs.conversation_upstream')}</span>
          </div>
          <div className={styles.conversationInlineValue} title={entry.upstream_url}>
            {entry.upstream_url}
          </div>
        </div>
      )}
    </div>
  );
}

export function ConversationLogsPanel() {
  const { t } = useTranslation();
  const { showNotification } = useNotificationStore();
  const connectionStatus = useAuthStore((state) => state.connectionStatus);
  const [filters, setFilters] = useState<ConversationLogFilters>(DEFAULT_FILTERS);
  const [appliedFilters, setAppliedFilters] = useState<ConversationLogFilters>(DEFAULT_FILTERS);
  const [entries, setEntries] = useState<ConversationLogSummary[]>([]);
  const [selectedId, setSelectedId] = useState('');
  const [selectedEntry, setSelectedEntry] = useState<ConversationLogEntry | null>(null);
  const [enabled, setEnabled] = useState(true);
  const [nextCursor, setNextCursor] = useState('');
  const [malformed, setMalformed] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState('');
  const [detailError, setDetailError] = useState('');

  const disableControls = connectionStatus !== 'connected';
  const filterSignature = useMemo(() => JSON.stringify(appliedFilters), [appliedFilters]);

  const hasDraftFilters = useMemo(
    () =>
      Object.entries(filters).some(([key, value]) => key !== 'limit' && value.trim() !== ''),
    [filters]
  );
  const selectedSummary = useMemo(
    () => entries.find((entry) => entry.id === selectedId),
    [entries, selectedId]
  );

  const loadEntries = async (cursor = '', append = false) => {
    if (connectionStatus !== 'connected') {
      setLoading(false);
      setLoadingMore(false);
      return;
    }
    if (append) {
      setLoadingMore(true);
    } else {
      setLoading(true);
    }
    setError('');
    try {
      const query = buildQuery(appliedFilters, cursor);
      const data = await logsApi.fetchConversationLogs(query);
      const nextEntries = Array.isArray(data.entries) ? data.entries : [];
      const combined = append ? [...entries, ...nextEntries] : nextEntries;
      setEnabled(Boolean(data.enabled));
      setEntries(combined);
      setNextCursor(data.next_cursor || '');
      setMalformed(data.malformed || 0);
      if (!selectedId || !combined.some((entry) => entry.id === selectedId)) {
        setSelectedId(combined[0]?.id || '');
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('logs.conversation_load_error');
      setError(message || t('logs.conversation_load_error'));
      if (!append) {
        setEntries([]);
        setSelectedId('');
      }
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  };

  useEffect(() => {
    void loadEntries('', false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [connectionStatus, filterSignature]);

  useEffect(() => {
    let active = true;
    if (!selectedId || connectionStatus !== 'connected') {
      setSelectedEntry(null);
      setDetailLoading(false);
      setDetailError('');
      return () => {
        active = false;
      };
    }

    setDetailLoading(true);
    setDetailError('');
    logsApi
      .fetchConversationLogDetail(selectedId)
      .then((data: ConversationLogDetailResponse) => {
        if (!active) return;
        setSelectedEntry(data.entry);
      })
      .catch((err: unknown) => {
        if (!active) return;
        const message = err instanceof Error ? err.message : t('logs.conversation_detail_error');
        setSelectedEntry(null);
        setDetailError(message || t('logs.conversation_detail_error'));
      })
      .finally(() => {
        if (active) setDetailLoading(false);
      });

    return () => {
      active = false;
    };
  }, [connectionStatus, selectedId, t]);

  const updateFilter = (key: keyof ConversationLogFilters, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const applyFilters = () => {
    setAppliedFilters({ ...filters });
    setSelectedId('');
  };

  const resetFilters = () => {
    setFilters(DEFAULT_FILTERS);
    setAppliedFilters(DEFAULT_FILTERS);
    setSelectedId('');
  };

  const copySelectedEntry = async () => {
    if (!selectedEntry) return;
    const ok = await copyToClipboard(JSON.stringify(selectedEntry, null, 2));
    showNotification(
      ok ? t('logs.conversation_copy_success') : t('logs.conversation_copy_failed'),
      ok ? 'success' : 'error'
    );
  };

  const downloadSelectedEntry = () => {
    if (!selectedEntry) return;
    const blob = new Blob([JSON.stringify(selectedEntry, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `conversation-log-${selectedEntry.id}.json`;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  };

  return (
    <Card className={styles.conversationPanel}>
      {error && <div className="error-box">{error}</div>}

      <div className={styles.conversationToolbar}>
        <div className={styles.conversationFilterGrid}>
          <Input
            label={t('logs.conversation_request_id')}
            value={filters.requestId}
            onChange={(event) => updateFilter('requestId', event.target.value)}
            placeholder="req"
          />
          <Input
            label={t('logs.conversation_provider')}
            value={filters.provider}
            onChange={(event) => updateFilter('provider', event.target.value)}
            placeholder="codex"
          />
          <Input
            label={t('logs.conversation_model')}
            value={filters.model}
            onChange={(event) => updateFilter('model', event.target.value)}
            placeholder="gpt"
          />
          <Input
            label={t('logs.conversation_path')}
            value={filters.path}
            onChange={(event) => updateFilter('path', event.target.value)}
            placeholder="/v1/chat"
          />
          <Input
            label={t('logs.conversation_status')}
            type="number"
            min={0}
            max={999}
            value={filters.statusCode}
            onChange={(event) => updateFilter('statusCode', event.target.value)}
            placeholder="200"
          />
          <div className={styles.conversationSelectField}>
            <label>{t('logs.conversation_error_state')}</label>
            <Select
              value={filters.hasError}
              options={[
                { value: '', label: t('logs.conversation_error_all') },
                { value: 'true', label: t('logs.conversation_error_only') },
                { value: 'false', label: t('logs.conversation_success_only') },
              ]}
              onChange={(value) => updateFilter('hasError', value)}
              ariaLabel={t('logs.conversation_error_state')}
            />
          </div>
          <Input
            label={t('logs.conversation_from')}
            type="datetime-local"
            value={filters.from}
            onChange={(event) => updateFilter('from', event.target.value)}
          />
          <Input
            label={t('logs.conversation_to')}
            type="datetime-local"
            value={filters.to}
            onChange={(event) => updateFilter('to', event.target.value)}
          />
          <Input
            label={t('logs.conversation_limit')}
            type="number"
            min={1}
            max={500}
            value={filters.limit}
            onChange={(event) => updateFilter('limit', event.target.value)}
          />
        </div>

        <div className={styles.conversationActions}>
          <Button
            variant="secondary"
            size="sm"
            onClick={applyFilters}
            disabled={disableControls || loading}
          >
            <span className={styles.buttonContent}>
              <IconSearch size={15} />
              {t('logs.conversation_apply_filters')}
            </span>
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={resetFilters}
            disabled={disableControls || loading || !hasDraftFilters}
          >
            <span className={styles.buttonContent}>
              <IconX size={15} />
              {t('logs.clear_filters')}
            </span>
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => void loadEntries('', false)}
            loading={loading}
            disabled={disableControls}
          >
            <span className={styles.buttonContent}>
              <IconRefreshCw size={15} />
              {t('common.refresh')}
            </span>
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={downloadSelectedEntry}
            disabled={!selectedEntry}
          >
            <span className={styles.buttonContent}>
              <IconDownload size={15} />
              {t('logs.conversation_download_json')}
            </span>
          </Button>
        </div>
      </div>

      <div className={styles.conversationStatusBar}>
        <span className={enabled ? styles.conversationOkBadge : styles.conversationWarnBadge}>
          {enabled ? t('logs.conversation_enabled') : t('logs.conversation_disabled')}
        </span>
        <span>{t('logs.conversation_entries_loaded', { count: entries.length })}</span>
        {nextCursor && <span>{t('logs.conversation_has_more')}</span>}
        {malformed > 0 && <span>{t('logs.conversation_malformed', { count: malformed })}</span>}
      </div>

      {!enabled ? (
        <EmptyState
          title={t('logs.conversation_disabled_title')}
          description={t('logs.conversation_disabled_desc')}
        />
      ) : loading && entries.length === 0 ? (
        <div className="hint">{t('common.loading')}</div>
      ) : entries.length === 0 ? (
        <EmptyState
          title={t('logs.conversation_empty_title')}
          description={t('logs.conversation_empty_desc')}
        />
      ) : (
        <div className={styles.conversationShell}>
          <div className={styles.conversationListPanel}>
            <div className={styles.conversationListHeader}>
              <span>{t('logs.conversation_list_title')}</span>
              <span>{entries.length}</span>
            </div>
            <div className={styles.conversationList}>
              {entries.map((entry) => (
                <SummaryRow
                  key={entry.id}
                  entry={entry}
                  active={entry.id === selectedId}
                  onSelect={() => setSelectedId(entry.id)}
                />
              ))}
            </div>
            {nextCursor && (
              <Button
                variant="secondary"
                size="sm"
                onClick={() => void loadEntries(nextCursor, true)}
                loading={loadingMore}
                fullWidth
              >
                {t('logs.conversation_load_more')}
              </Button>
            )}
          </div>

          <div className={styles.conversationDetailPanel}>
            <div className={styles.conversationListHeader}>
              <span>{t('logs.conversation_detail_title')}</span>
              <IconFileText size={16} />
            </div>
            <ConversationDetail
              entry={selectedEntry}
              summary={selectedSummary}
              loading={detailLoading}
              error={detailError}
              onCopy={copySelectedEntry}
            />
          </div>
        </div>
      )}
    </Card>
  );
}
