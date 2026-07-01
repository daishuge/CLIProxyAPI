import { type ChangeEvent, type CSSProperties, type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import {
  IconChartLine,
  IconDiamond,
  IconDollarSign,
  IconDownload,
  IconKey,
  IconModelCluster,
  IconRefreshCw,
  IconSatellite,
  IconShield,
  IconTimer,
  IconTrendingUp,
  IconUpload,
} from '@/components/ui/icons';
import pricingConfigRaw from '@/data/openaiModelPricing.json?raw';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { usageApi } from '@/services/api';
import { useAuthStore, useNotificationStore } from '@/stores';
import type {
  UsageApiSnapshot,
  UsageExportPayload,
  UsageRequestDetail,
  UsageStatisticsResponse,
  UsageStatisticsSnapshot,
  UsageTokenStats,
} from '@/types';
import styles from './UsagePage.module.scss';

type BreakdownTab = 'models' | 'keys' | 'apis' | 'details';
type TrendSpan = '24h' | '7d' | '30d' | 'all';
type TrendMetric = 'requests' | 'tokens' | 'cost';

interface PricingRate {
  input: number;
  cached_input: number;
  output: number;
  aliases?: string[];
}

interface PricingConfig {
  currency: string;
  unit: string;
  source: string;
  updated_at: string;
  models: Record<string, PricingRate>;
}

interface DetailRecord {
  id: string;
  apiName: string;
  modelName: string;
  keyName: string;
  timestampMs: number;
  detail: UsageRequestDetail;
  cost: number | null;
  pricingModel: string | null;
}

interface AggregateRow {
  id: string;
  label: string;
  apiName?: string;
  keyName?: string;
  requests: number;
  failed: number;
  tokens: number;
  inputTokens: number;
  outputTokens: number;
  reasoningTokens: number;
  cachedTokens: number;
  totalLatencyMs: number;
  latencySamples: number;
  totalFirstByteMs: number;
  firstByteSamples: number;
  estimatedCost: number | null;
  pricedRequests: number;
  lastSeenMs: number;
  models: Set<string>;
  keys: Set<string>;
}

interface TrendPoint {
  key: string;
  label: string;
  requests: number;
  failures: number;
  tokens: number;
  cost: number;
}

interface ActivityWindowSummary {
  key: '1h' | '4h' | '8h' | '24h';
  hours: number;
  requests: number;
  failures: number;
  tokens: number;
  cost: number;
  rpm: number;
  tpm: number;
  averageLatencyMs: number;
}

interface HourActivityBucket {
  key: string;
  label: string;
  requests: number;
  failures: number;
  tokens: number;
  cost: number;
}

interface TokenBreakdownItem {
  key: 'input' | 'cached' | 'output' | 'reasoning';
  label: string;
  value: number;
  className: string;
}

const pricingConfig = JSON.parse(pricingConfigRaw) as PricingConfig;

const emptyUsage: UsageStatisticsSnapshot = {
  total_requests: 0,
  success_count: 0,
  failure_count: 0,
  total_tokens: 0,
  total_input_tokens: 0,
  total_cached_tokens: 0,
  cache_hit_rate: 0,
  average_latency_ms: 0,
  average_first_byte_latency_ms: 0,
  tps: 0,
  apis: {},
  requests_by_day: {},
  requests_by_hour: {},
  tokens_by_day: {},
  tokens_by_hour: {},
};

const pricingAliases = (() => {
  const entries: Array<[string, string, PricingRate]> = [];
  Object.entries(pricingConfig.models).forEach(([model, rate]) => {
    entries.push([model.toLowerCase(), model, rate]);
    (rate.aliases ?? []).forEach((alias) => entries.push([alias.toLowerCase(), model, rate]));
  });
  return entries.sort((a, b) => b[0].length - a[0].length);
})();

const safeNumber = (value: number | null | undefined): number =>
  typeof value === 'number' && Number.isFinite(value) ? value : 0;

const thinkingLevelSuffixes = ['low', 'medium', 'high', 'xhigh'];

const isRecord = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === 'object' && !Array.isArray(value);

const detailTimestamp = (detail: UsageRequestDetail): number => {
  const timestamp = Date.parse(detail.timestamp);
  return Number.isFinite(timestamp) ? timestamp : 0;
};

const normalizeModelName = (modelName: string): string[] => {
  const raw = modelName.trim().toLowerCase();
  const variants = new Set<string>();
  const addVariant = (value: string | undefined) => {
    const normalized = (value ?? '').trim().toLowerCase();
    if (!normalized) return;
    variants.add(normalized);
    const parenthesized = normalized.match(/^(.*)\((low|medium|high|xhigh)\)$/);
    if (parenthesized?.[1]) variants.add(parenthesized[1]);
    thinkingLevelSuffixes.forEach((suffix) => {
      const marker = `-${suffix}`;
      if (normalized.endsWith(marker)) {
        variants.add(normalized.slice(0, -marker.length));
      }
    });
  };
  addVariant(raw);
  addVariant(raw.split('/').pop());
  addVariant(raw.split(':').pop());
  return Array.from(variants);
};

const findPricing = (modelName: string): { model: string; rate: PricingRate } | null => {
  const variants = normalizeModelName(modelName);
  for (const candidate of variants) {
    const exact = pricingAliases.find(([alias]) => alias === candidate);
    if (exact) return { model: exact[1], rate: exact[2] };
  }
  for (const candidate of variants) {
    const prefix = pricingAliases.find(([alias]) => candidate.startsWith(`${alias}-`));
    if (prefix) return { model: prefix[1], rate: prefix[2] };
  }
  return null;
};

const calculateCost = (modelName: string, tokens: UsageTokenStats | undefined) => {
  const pricing = findPricing(modelName);
  if (!pricing || !tokens) return { cost: null, pricingModel: null };

  const inputTokens = safeNumber(tokens.input_tokens);
  const cachedTokens = safeNumber(tokens.cached_tokens);
  const outputTokens = safeNumber(tokens.output_tokens);
  const totalTokens = safeNumber(tokens.total_tokens);
  const inferredOutputTokens = Math.max(outputTokens, totalTokens - inputTokens);
  const uncachedInputTokens = Math.max(inputTokens - cachedTokens, 0);
  const cost =
    (uncachedInputTokens / 1_000_000) * pricing.rate.input +
    (cachedTokens / 1_000_000) * pricing.rate.cached_input +
    (Math.max(inferredOutputTokens, 0) / 1_000_000) * pricing.rate.output;
  return { cost, pricingModel: pricing.model };
};

const newAggregateRow = (id: string, label: string, extra?: Partial<AggregateRow>): AggregateRow => ({
  id,
  label,
  requests: 0,
  failed: 0,
  tokens: 0,
  inputTokens: 0,
  outputTokens: 0,
  reasoningTokens: 0,
  cachedTokens: 0,
  totalLatencyMs: 0,
  latencySamples: 0,
  totalFirstByteMs: 0,
  firstByteSamples: 0,
  estimatedCost: null,
  pricedRequests: 0,
  lastSeenMs: 0,
  models: new Set<string>(),
  keys: new Set<string>(),
  ...extra,
});

const addDetailToAggregate = (row: AggregateRow, record: DetailRecord) => {
  const tokens = record.detail.tokens ?? {
    input_tokens: 0,
    output_tokens: 0,
    reasoning_tokens: 0,
    cached_tokens: 0,
    total_tokens: 0,
  };
  row.requests += 1;
  if (record.detail.failed) row.failed += 1;
  row.tokens += safeNumber(tokens.total_tokens);
  row.inputTokens += safeNumber(tokens.input_tokens);
  row.outputTokens += safeNumber(tokens.output_tokens);
  row.reasoningTokens += safeNumber(tokens.reasoning_tokens);
  row.cachedTokens += safeNumber(tokens.cached_tokens);
  if (safeNumber(record.detail.latency_ms) > 0) {
    row.totalLatencyMs += safeNumber(record.detail.latency_ms);
    row.latencySamples += 1;
  }
  if (safeNumber(record.detail.first_byte_latency_ms) > 0) {
    row.totalFirstByteMs += safeNumber(record.detail.first_byte_latency_ms);
    row.firstByteSamples += 1;
  }
  if (record.cost !== null) {
    row.estimatedCost = (row.estimatedCost ?? 0) + record.cost;
    row.pricedRequests += 1;
  }
  row.lastSeenMs = Math.max(row.lastSeenMs, record.timestampMs);
  row.models.add(record.modelName);
  row.keys.add(record.keyName);
};

const average = (total: number, samples: number): number => (samples > 0 ? total / samples : 0);

const successRate = (row: Pick<AggregateRow, 'requests' | 'failed'>): number =>
  row.requests > 0 ? ((row.requests - row.failed) / row.requests) * 100 : 0;

const flattenDetails = (usage: UsageStatisticsSnapshot): DetailRecord[] =>
  Object.entries(usage.apis ?? {}).flatMap(([apiName, api]) =>
    Object.entries(api.models ?? {}).flatMap(([modelName, model]) =>
      (model.details ?? []).map((detail, index) => {
        const keyName = detail.auth_index || detail.source || apiName || '-';
        const { cost, pricingModel } = calculateCost(modelName, detail.tokens);
        return {
          id: `${apiName}:${modelName}:${detail.timestamp}:${index}`,
          apiName,
          modelName,
          keyName,
          timestampMs: detailTimestamp(detail),
          detail,
          cost,
          pricingModel,
        };
      })
    )
  );

const aggregateBy = (
  records: DetailRecord[],
  resolve: (record: DetailRecord) => { id: string; label: string; extra?: Partial<AggregateRow> }
) => {
  const rows = new Map<string, AggregateRow>();
  records.forEach((record) => {
    const target = resolve(record);
    const existing = rows.get(target.id);
    const row = existing ?? newAggregateRow(target.id, target.label, target.extra);
    addDetailToAggregate(row, record);
    rows.set(target.id, row);
  });
  return Array.from(rows.values()).sort((a, b) => b.requests - a.requests);
};

const apiSnapshotRows = (apis: Record<string, UsageApiSnapshot> | undefined): AggregateRow[] =>
  Object.entries(apis ?? {})
    .map(([apiName, api]) => {
      const row = newAggregateRow(apiName, apiName, { apiName });
      row.requests = safeNumber(api.total_requests);
      row.tokens = safeNumber(api.total_tokens);
      row.inputTokens = safeNumber(api.total_input_tokens);
      row.cachedTokens = safeNumber(api.total_cached_tokens);
      row.totalLatencyMs = safeNumber(api.average_latency_ms) * row.requests;
      row.latencySamples = row.requests;
      row.totalFirstByteMs = safeNumber(api.average_first_byte_latency_ms) * row.requests;
      row.firstByteSamples = row.requests;
      Object.keys(api.models ?? {}).forEach((model) => row.models.add(model));
      row.failed = Object.values(api.models ?? {}).reduce(
        (total, model) => total + (model.details ?? []).filter((detail) => detail.failed).length,
        0
      );
      return row;
    })
    .sort((a, b) => b.requests - a.requests);

const createDateKey = (date: Date): string =>
  `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(
    date.getDate()
  ).padStart(2, '0')}`;

const createHourKey = (date: Date): string => `${createDateKey(date)} ${String(date.getHours()).padStart(2, '0')}:00`;

const createHourLabel = (date: Date): string => `${String(date.getHours()).padStart(2, '0')}:00`;

const activityWindowDefinitions: ReadonlyArray<Pick<ActivityWindowSummary, 'key' | 'hours'>> = [
  { key: '1h', hours: 1 },
  { key: '4h', hours: 4 },
  { key: '8h', hours: 8 },
  { key: '24h', hours: 24 },
];

const summarizeActivityWindow = (
  records: DetailRecord[],
  windowDefinition: Pick<ActivityWindowSummary, 'key' | 'hours'>,
  nowMs: number
): ActivityWindowSummary => {
  const startMs = nowMs - windowDefinition.hours * 60 * 60 * 1000;
  const scoped = records.filter((record) => record.timestampMs > 0 && record.timestampMs >= startMs);
  const requests = scoped.length;
  const failures = scoped.filter((record) => record.detail.failed).length;
  const tokens = scoped.reduce((total, record) => total + safeNumber(record.detail.tokens?.total_tokens), 0);
  const cost = scoped.reduce((total, record) => total + (record.cost ?? 0), 0);
  const latencySamples = scoped.filter((record) => safeNumber(record.detail.latency_ms) > 0);
  const minutes = Math.max(windowDefinition.hours * 60, 1);

  return {
    ...windowDefinition,
    requests,
    failures,
    tokens,
    cost,
    rpm: requests / minutes,
    tpm: tokens / minutes,
    averageLatencyMs: average(
      latencySamples.reduce((total, record) => total + safeNumber(record.detail.latency_ms), 0),
      latencySamples.length
    ),
  };
};

const makeRecentHourBuckets = (records: DetailRecord[], hours: number, nowMs: number): HourActivityBucket[] => {
  const buckets = new Map<string, HourActivityBucket>();
  for (let i = hours - 1; i >= 0; i -= 1) {
    const date = new Date(nowMs - i * 60 * 60 * 1000);
    const key = createHourKey(date);
    buckets.set(key, {
      key,
      label: createHourLabel(date),
      requests: 0,
      failures: 0,
      tokens: 0,
      cost: 0,
    });
  }

  const startMs = nowMs - hours * 60 * 60 * 1000;
  records
    .filter((record) => record.timestampMs > 0 && record.timestampMs >= startMs)
    .forEach((record) => {
      const date = new Date(record.timestampMs);
      const key = createHourKey(date);
      const bucket =
        buckets.get(key) ??
        {
          key,
          label: createHourLabel(date),
          requests: 0,
          failures: 0,
          tokens: 0,
          cost: 0,
        };
      bucket.requests += 1;
      bucket.failures += record.detail.failed ? 1 : 0;
      bucket.tokens += safeNumber(record.detail.tokens?.total_tokens);
      bucket.cost += record.cost ?? 0;
      buckets.set(key, bucket);
    });

  return Array.from(buckets.values()).sort((a, b) => a.key.localeCompare(b.key));
};

const summarizeTokenBreakdown = (records: DetailRecord[], usage: UsageStatisticsSnapshot) => {
  if (records.length) {
    return records.reduce(
      (totals, record) => {
        const tokens = record.detail.tokens;
        const inputTokens = safeNumber(tokens?.input_tokens);
        const cachedTokens = safeNumber(tokens?.cached_tokens);
        totals.inputTokens += Math.max(inputTokens - cachedTokens, 0);
        totals.cachedTokens += cachedTokens;
        totals.outputTokens += safeNumber(tokens?.output_tokens);
        totals.reasoningTokens += safeNumber(tokens?.reasoning_tokens);
        return totals;
      },
      { inputTokens: 0, cachedTokens: 0, outputTokens: 0, reasoningTokens: 0 }
    );
  }

  const inputTokens = safeNumber(usage.total_input_tokens);
  const cachedTokens = safeNumber(usage.total_cached_tokens);
  return {
    inputTokens: Math.max(inputTokens - cachedTokens, 0),
    cachedTokens,
    outputTokens: Math.max(safeNumber(usage.total_tokens) - inputTokens, 0),
    reasoningTokens: 0,
  };
};

const findPeakBucket = (
  records: DetailRecord[],
  usage: UsageStatisticsSnapshot
): HourActivityBucket | null => {
  if (records.length) {
    const buckets = new Map<string, HourActivityBucket>();
    records.forEach((record) => {
      if (record.timestampMs <= 0) return;
      const date = new Date(record.timestampMs);
      const key = createHourKey(date);
      const bucket =
        buckets.get(key) ??
        {
          key,
          label: key.slice(5),
          requests: 0,
          failures: 0,
          tokens: 0,
          cost: 0,
        };
      bucket.requests += 1;
      bucket.failures += record.detail.failed ? 1 : 0;
      bucket.tokens += safeNumber(record.detail.tokens?.total_tokens);
      bucket.cost += record.cost ?? 0;
      buckets.set(key, bucket);
    });
    return Array.from(buckets.values()).sort((a, b) => b.requests - a.requests)[0] ?? null;
  }

  const requestHours = usage.requests_by_hour ?? {};
  return (
    Object.entries(requestHours)
      .map(([hour, requests]) => ({
        key: hour,
        label: `${hour}:00`,
        requests: safeNumber(requests),
        failures: 0,
        tokens: safeNumber(usage.tokens_by_hour?.[hour]),
        cost: 0,
      }))
      .sort((a, b) => b.requests - a.requests)[0] ?? null
  );
};

const trendLabel = (key: string, span: TrendSpan): string => {
  if (span === '24h') return key.slice(11);
  if (span === 'all' && key.length === 7) return key;
  return key.slice(5);
};

const makeTrendBuckets = (
  records: DetailRecord[],
  usage: UsageStatisticsSnapshot,
  span: TrendSpan
): TrendPoint[] => {
  const now = new Date();
  const buckets = new Map<string, TrendPoint>();
  const addBucket = (key: string) => {
    if (!buckets.has(key)) {
      buckets.set(key, { key, label: trendLabel(key, span), requests: 0, failures: 0, tokens: 0, cost: 0 });
    }
    return buckets.get(key)!;
  };

  if (span === '24h') {
    for (let i = 23; i >= 0; i -= 1) {
      const date = new Date(now.getTime() - i * 60 * 60 * 1000);
      addBucket(createHourKey(date));
    }
  } else if (span === '7d' || span === '30d') {
    const days = span === '7d' ? 6 : 29;
    for (let i = days; i >= 0; i -= 1) {
      const date = new Date(now.getFullYear(), now.getMonth(), now.getDate() - i);
      addBucket(createDateKey(date));
    }
  }

  const startMs =
    span === '24h'
      ? now.getTime() - 24 * 60 * 60 * 1000
      : span === '7d'
        ? now.getTime() - 7 * 24 * 60 * 60 * 1000
        : span === '30d'
          ? now.getTime() - 30 * 24 * 60 * 60 * 1000
          : 0;

  records
    .filter((record) => record.timestampMs > 0 && record.timestampMs >= startMs)
    .forEach((record) => {
      const date = new Date(record.timestampMs);
      const key =
        span === '24h'
          ? createHourKey(date)
          : span === 'all' && records.length > 1200
            ? `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
            : createDateKey(date);
      const bucket = addBucket(key);
      bucket.requests += 1;
      bucket.failures += record.detail.failed ? 1 : 0;
      bucket.tokens += safeNumber(record.detail.tokens?.total_tokens);
      bucket.cost += record.cost ?? 0;
    });

  if (records.length === 0 && span !== '24h') {
    const requestMap = usage.requests_by_day ?? {};
    const tokenMap = usage.tokens_by_day ?? {};
    Object.entries(requestMap).forEach(([key, value]) => {
      const bucket = addBucket(key);
      bucket.requests = safeNumber(value);
      bucket.tokens = safeNumber(tokenMap[key]);
    });
  }

  return Array.from(buckets.values()).sort((a, b) => a.key.localeCompare(b.key));
};

function TrendChart({
  rows,
  metric,
  formatValue,
  emptyLabel,
}: {
  rows: TrendPoint[];
  metric: TrendMetric;
  formatValue: (value: number) => string;
  emptyLabel: string;
}) {
  const values = rows.map((row) => (metric === 'requests' ? row.requests : metric === 'tokens' ? row.tokens : row.cost));
  const max = Math.max(...values, 0);
  const chartWidth = 720;
  const chartHeight = 220;
  const paddingX = 28;
  const paddingY = 24;
  const plotWidth = chartWidth - paddingX * 2;
  const plotHeight = chartHeight - paddingY * 2;
  const points = values.map((value, index) => {
    const x = paddingX + (rows.length <= 1 ? plotWidth / 2 : (index / (rows.length - 1)) * plotWidth);
    const y = paddingY + plotHeight - (max > 0 ? (value / max) * plotHeight : 0);
    return { x, y, value, row: rows[index]! };
  });
  const path = points.map((point) => `${point.x},${point.y}`).join(' ');
  const areaPath = points.length
    ? `${paddingX},${chartHeight - paddingY} ${path} ${chartWidth - paddingX},${chartHeight - paddingY}`
    : '';
  const visibleLabels = points.filter((_, index) => {
    if (rows.length <= 8) return true;
    return index === 0 || index === rows.length - 1 || index % Math.ceil(rows.length / 6) === 0;
  });

  if (!rows.length) {
    return <div className={styles.emptyInline}>{emptyLabel}</div>;
  }

  return (
    <div className={styles.chartFrame}>
      <svg className={styles.lineChart} viewBox={`0 0 ${chartWidth} ${chartHeight}`} role="img">
        <polygon className={styles.chartArea} points={areaPath} />
        <polyline className={styles.chartLine} points={path} />
        {points.map((point) => (
          <circle key={point.row.key} className={styles.chartPoint} cx={point.x} cy={point.y} r="3.5">
            <title>
              {point.row.key} · {formatValue(point.value)}
            </title>
          </circle>
        ))}
        {visibleLabels.map((point) => (
          <text key={point.row.key} className={styles.chartLabel} x={point.x} y={chartHeight - 5} textAnchor="middle">
            {point.row.label}
          </text>
        ))}
      </svg>
    </div>
  );
}

export function UsagePage() {
  const { t, i18n } = useTranslation();
  const connectionStatus = useAuthStore((state) => state.connectionStatus);
  const { showNotification } = useNotificationStore();

  const [response, setResponse] = useState<UsageStatisticsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState('');
  const [activeTab, setActiveTab] = useState<BreakdownTab>('models');
  const [trendSpan, setTrendSpan] = useState<TrendSpan>('7d');
  const [trendMetric, setTrendMetric] = useState<TrendMetric>('requests');
  const [snapshotNowMs, setSnapshotNowMs] = useState(() => Date.now());

  const importInputRef = useRef<HTMLInputElement | null>(null);

  const usage = response?.usage ?? emptyUsage;
  const detailRecords = useMemo(() => flattenDetails(usage), [usage]);
  const failedRequests = safeNumber(response?.failed_requests ?? usage.failure_count);
  const totalOutputTokens = useMemo(
    () => detailRecords.reduce((total, record) => total + safeNumber(record.detail.tokens?.output_tokens), 0),
    [detailRecords]
  );
  const estimatedTotalCost = useMemo(() => {
    const priced = detailRecords.filter((record) => record.cost !== null);
    if (!priced.length) return null;
    return priced.reduce((total, record) => total + (record.cost ?? 0), 0);
  }, [detailRecords]);
  const pricedRequestCount = detailRecords.filter((record) => record.cost !== null).length;
  const hasUsage =
    safeNumber(usage.total_requests) > 0 ||
    Object.keys(usage.apis ?? {}).length > 0 ||
    detailRecords.length > 0;

  const numberFormatter = useMemo(
    () => new Intl.NumberFormat(i18n.language || undefined),
    [i18n.language]
  );
  const compactNumberFormatter = useMemo(
    () => new Intl.NumberFormat(i18n.language || undefined, { notation: 'compact', maximumFractionDigits: 1 }),
    [i18n.language]
  );
  const currencyFormatter = useMemo(
    () =>
      new Intl.NumberFormat(i18n.language || undefined, {
        style: 'currency',
        currency: pricingConfig.currency || 'USD',
        maximumFractionDigits: 4,
      }),
    [i18n.language]
  );

  const formatNumber = useCallback(
    (value: number | null | undefined) => numberFormatter.format(safeNumber(value)),
    [numberFormatter]
  );
  const formatCompact = useCallback(
    (value: number | null | undefined) => compactNumberFormatter.format(safeNumber(value)),
    [compactNumberFormatter]
  );
  const formatPercent = useCallback((value: number | null | undefined) => {
    const numeric = safeNumber(value);
    return `${numeric.toFixed(numeric >= 10 ? 1 : 2)}%`;
  }, []);
  const formatLatency = useCallback(
    (value: number | null | undefined) => `${formatNumber(Math.round(safeNumber(value)))} ms`,
    [formatNumber]
  );
  const formatCost = useCallback(
    (value: number | null | undefined) => (value === null || value === undefined ? '-' : currencyFormatter.format(value)),
    [currencyFormatter]
  );
  const formatRate = useCallback(
    (value: number | null | undefined) => {
      const numeric = safeNumber(value);
      if (numeric >= 100) return numberFormatter.format(Math.round(numeric));
      if (numeric >= 10) return numeric.toFixed(1);
      return numeric.toFixed(2);
    },
    [numberFormatter]
  );
  const formatTimestamp = useCallback(
    (timestampMs: number) => {
      if (!timestampMs) return '-';
      return new Date(timestampMs).toLocaleString(i18n.language || undefined);
    },
    [i18n.language]
  );

  const apiRows = useMemo(() => {
    const detailRows = aggregateBy(detailRecords, (record) => ({
      id: record.apiName,
      label: record.apiName,
      extra: { apiName: record.apiName },
    }));
    return detailRows.length ? detailRows : apiSnapshotRows(usage.apis);
  }, [detailRecords, usage.apis]);

  const modelRows = useMemo(
    () =>
      aggregateBy(detailRecords, (record) => ({
        id: record.modelName,
        label: record.modelName,
        extra: { apiName: record.apiName },
      })),
    [detailRecords]
  );

  const keyRows = useMemo(
    () =>
      aggregateBy(detailRecords, (record) => ({
        id: record.keyName,
        label: record.keyName,
        extra: { keyName: record.keyName, apiName: record.apiName },
      })),
    [detailRecords]
  );

  const recentRows = useMemo(
    () => [...detailRecords].sort((a, b) => b.timestampMs - a.timestampMs).slice(0, 80),
    [detailRecords]
  );

  const nowMs = snapshotNowMs;
  const activityWindowSummaries = useMemo(
    () => activityWindowDefinitions.map((definition) => summarizeActivityWindow(detailRecords, definition, nowMs)),
    [detailRecords, nowMs]
  );
  const recentHourBuckets = useMemo(() => makeRecentHourBuckets(detailRecords, 24, nowMs), [detailRecords, nowMs]);
  const tokenBreakdown = useMemo(() => summarizeTokenBreakdown(detailRecords, usage), [detailRecords, usage]);
  const peakBucket = useMemo(() => findPeakBucket(detailRecords, usage), [detailRecords, usage]);

  const trendRows = useMemo(
    () => makeTrendBuckets(detailRecords, usage, trendSpan),
    [detailRecords, trendSpan, usage]
  );

  const topModel = modelRows[0]?.label ?? '-';
  const topKey = keyRows[0]?.label ?? '-';
  const last24hSummary = activityWindowSummaries.find((summary) => summary.key === '24h');
  const failedRate = safeNumber(usage.total_requests) > 0 ? (failedRequests / safeNumber(usage.total_requests)) * 100 : 0;
  const lastSeenMs = detailRecords.reduce((latest, record) => Math.max(latest, record.timestampMs), 0);
  const activeDays =
    Object.keys(usage.requests_by_day ?? {}).length ||
    new Set(detailRecords.map((record) => (record.timestampMs > 0 ? createDateKey(new Date(record.timestampMs)) : '')).filter(Boolean)).size;
  const activeHours =
    detailRecords.length > 0
      ? new Set(detailRecords.map((record) => (record.timestampMs > 0 ? createHourKey(new Date(record.timestampMs)) : '')).filter(Boolean)).size
      : Object.keys(usage.requests_by_hour ?? {}).length;
  const tokenBreakdownTotal =
    tokenBreakdown.inputTokens +
    tokenBreakdown.cachedTokens +
    tokenBreakdown.outputTokens +
    tokenBreakdown.reasoningTokens;
  const recentActivityMax = Math.max(...recentHourBuckets.map((bucket) => bucket.requests), 1);
  const tokenBreakdownItems = useMemo<TokenBreakdownItem[]>(
    () => [
      {
        key: 'input',
        label: t('usage_statistics.uncached_input_tokens'),
        value: tokenBreakdown.inputTokens,
        className: styles.tokenInput,
      },
      {
        key: 'cached',
        label: t('usage_statistics.cached_tokens'),
        value: tokenBreakdown.cachedTokens,
        className: styles.tokenCached,
      },
      {
        key: 'output',
        label: t('usage_statistics.output_tokens'),
        value: tokenBreakdown.outputTokens,
        className: styles.tokenOutput,
      },
      {
        key: 'reasoning',
        label: t('usage_statistics.reasoning_tokens'),
        value: tokenBreakdown.reasoningTokens,
        className: styles.tokenReasoning,
      },
    ],
    [t, tokenBreakdown]
  );
  const topDimensionGroups = useMemo(
    () => [
      { key: 'models', title: t('usage_statistics.top_models'), rows: modelRows.slice(0, 5) },
      { key: 'apis', title: t('usage_statistics.top_apis'), rows: apiRows.slice(0, 5) },
      { key: 'keys', title: t('usage_statistics.top_keys'), rows: keyRows.slice(0, 5) },
    ],
    [apiRows, keyRows, modelRows, t]
  );

  const loadUsage = useCallback(async () => {
    if (connectionStatus !== 'connected') {
      setLoading(false);
      setResponse(null);
      setSnapshotNowMs(Date.now());
      setError(t('usage_statistics.connection_required'));
      return;
    }

    setLoading(true);
    setError('');
    try {
      const data = await usageApi.getStatistics();
      setResponse(data);
      setSnapshotNowMs(Date.now());
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('notification.refresh_failed');
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [connectionStatus, t]);

  useHeaderRefresh(loadUsage);

  useEffect(() => {
    loadUsage();
  }, [loadUsage]);

  const handleExport = async () => {
    setExporting(true);
    try {
      const payload = await usageApi.exportStatistics();
      const exportedAt = payload.exported_at || new Date().toISOString();
      const fileSafeDate = exportedAt.replace(/[:.]/g, '-');
      const blob = new Blob([JSON.stringify(payload, null, 2)], {
        type: 'application/json;charset=utf-8',
      });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = `ppap-usage-${fileSafeDate}.json`;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);
      showNotification(t('usage_statistics.export_success'), 'success');
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('usage_statistics.export_failed');
      showNotification(message, 'error');
    } finally {
      setExporting(false);
    }
  };

  const normalizeImportPayload = (parsed: unknown): UsageExportPayload => {
    if (!isRecord(parsed)) {
      throw new Error(t('usage_statistics.import_invalid'));
    }

    const usageValue = isRecord(parsed.usage) ? parsed.usage : parsed;
    if (!isRecord(usageValue)) {
      throw new Error(t('usage_statistics.import_invalid'));
    }

    return {
      version: typeof parsed.version === 'number' ? parsed.version : 1,
      exported_at: typeof parsed.exported_at === 'string' ? parsed.exported_at : new Date().toISOString(),
      usage: usageValue as unknown as UsageStatisticsSnapshot,
    };
  };

  const handleImportFile = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.currentTarget.files?.[0];
    event.currentTarget.value = '';
    if (!file) return;

    setImporting(true);
    try {
      const parsed = JSON.parse(await file.text());
      const payload = normalizeImportPayload(parsed);
      const result = await usageApi.importStatistics(payload);
      showNotification(
        t('usage_statistics.import_success', {
          added: result.added,
          skipped: result.skipped,
        }),
        'success'
      );
      await loadUsage();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('usage_statistics.import_failed');
      showNotification(message, 'error');
    } finally {
      setImporting(false);
    }
  };

  const renderMetric = (label: string, value: string, sublabel?: string, icon?: ReactNode) => (
    <div className={styles.metricTile}>
      <div className={styles.metricHeader}>
        <span className={styles.metricLabel}>{label}</span>
        {icon ? <span className={styles.metricIcon}>{icon}</span> : null}
      </div>
      <span className={styles.metricValue}>{value}</span>
      {sublabel && <span className={styles.metricSub}>{sublabel}</span>}
    </div>
  );

  const renderAggregateTable = (rows: AggregateRow[], mode: BreakdownTab) => (
    <div className={styles.tableScroll}>
      <table className={styles.table}>
        <thead>
          <tr>
            <th>{mode === 'keys' ? t('usage_statistics.key') : mode === 'apis' ? t('usage_statistics.api') : t('usage_statistics.model')}</th>
            <th>{t('usage_statistics.requests')}</th>
            <th>{t('usage_statistics.success_rate')}</th>
            <th>{t('usage_statistics.tokens')}</th>
            <th>{t('usage_statistics.input_tokens')}</th>
            <th>{t('usage_statistics.output_tokens')}</th>
            <th>{t('usage_statistics.cached_tokens')}</th>
            <th>{t('usage_statistics.estimated_cost')}</th>
            <th>{t('usage_statistics.first_byte')}</th>
            <th>{t('usage_statistics.latency')}</th>
            <th>{mode === 'models' ? t('usage_statistics.keys') : t('usage_statistics.models')}</th>
          </tr>
        </thead>
        <tbody>
          {rows.length ? (
            rows.map((row) => (
              <tr key={row.id}>
                <td className={styles.primaryCell}>
                  <span className={styles.monoCell}>{row.label}</span>
                  {row.apiName && mode !== 'apis' ? <span className={styles.cellHint}>{row.apiName}</span> : null}
                </td>
                <td>{formatNumber(row.requests)}</td>
                <td>{formatPercent(successRate(row))}</td>
                <td>{formatNumber(row.tokens)}</td>
                <td>{formatNumber(row.inputTokens)}</td>
                <td>{formatNumber(row.outputTokens)}</td>
                <td>{formatNumber(row.cachedTokens)}</td>
                <td>{formatCost(row.estimatedCost)}</td>
                <td>{formatLatency(average(row.totalFirstByteMs, row.firstByteSamples))}</td>
                <td>{formatLatency(average(row.totalLatencyMs, row.latencySamples))}</td>
                <td>{formatNumber(mode === 'models' ? row.keys.size : row.models.size)}</td>
              </tr>
            ))
          ) : (
            <tr>
              <td colSpan={11} className={styles.emptyCell}>
                {t('usage_statistics.no_rows')}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );

  const trendFormatter = (value: number) =>
    trendMetric === 'cost' ? formatCost(value) : trendMetric === 'tokens' ? formatCompact(value) : formatNumber(value);

  return (
    <div className={styles.container}>
      <div className={styles.pageHeader}>
        <div>
          <h1 className={styles.pageTitle}>{t('usage_statistics.title')}</h1>
          <p className={styles.description}>{t('usage_statistics.description')}</p>
          <div className={styles.priceSource}>
            <IconDollarSign size={14} />
            <span>
              {t('usage_statistics.pricing_source', {
                date: pricingConfig.updated_at,
                unit: pricingConfig.unit,
              })}
            </span>
          </div>
        </div>
        <div className={styles.headerActions}>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={loadUsage}
            loading={loading}
            disabled={connectionStatus !== 'connected'}
          >
            <IconRefreshCw size={16} />
            {t('common.refresh')}
          </Button>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={() => importInputRef.current?.click()}
            loading={importing}
            disabled={connectionStatus !== 'connected'}
          >
            <IconUpload size={16} />
            {t('usage_statistics.import')}
          </Button>
          <Button
            type="button"
            variant="primary"
            size="sm"
            onClick={handleExport}
            loading={exporting}
            disabled={connectionStatus !== 'connected'}
          >
            <IconDownload size={16} />
            {t('usage_statistics.export')}
          </Button>
          <input
            ref={importInputRef}
            className={styles.hiddenInput}
            type="file"
            accept="application/json,.json"
            onChange={handleImportFile}
          />
        </div>
      </div>

      {error && <div className={styles.errorBox}>{error}</div>}

      <section className={styles.metricGrid} aria-label={t('usage_statistics.summary')}>
        {renderMetric(t('usage_statistics.total_requests'), formatNumber(usage.total_requests), `${formatNumber(usage.success_count)} ${t('usage_statistics.success')} / ${formatNumber(failedRequests)} ${t('usage_statistics.failed')}`, <IconSatellite size={16} />)}
        {renderMetric(t('usage_statistics.estimated_cost'), formatCost(estimatedTotalCost), t('usage_statistics.priced_requests', { count: pricedRequestCount }), <IconDollarSign size={16} />)}
        {renderMetric(t('usage_statistics.cache_hit_rate'), formatPercent(usage.cache_hit_rate), `${formatNumber(usage.total_cached_tokens)} ${t('usage_statistics.cached_tokens')}`, <IconShield size={16} />)}
        {renderMetric(t('usage_statistics.first_byte_latency'), formatLatency(usage.average_first_byte_latency_ms), t('usage_statistics.average_value'), <IconTimer size={16} />)}
        {renderMetric(t('usage_statistics.average_latency'), formatLatency(usage.average_latency_ms), t('usage_statistics.average_value'), <IconTimer size={16} />)}
        {renderMetric(t('usage_statistics.tps'), safeNumber(usage.tps).toFixed(2), t('usage_statistics.total_throughput'), <IconTrendingUp size={16} />)}
        {renderMetric(t('usage_statistics.total_tokens'), formatNumber(usage.total_tokens), `${formatNumber(usage.total_input_tokens)} in / ${formatNumber(totalOutputTokens)} out`, <IconDiamond size={16} />)}
        {renderMetric(t('usage_statistics.recent_24h'), formatNumber(last24hSummary?.requests), t('usage_statistics.recent_window_sub', { tokens: formatCompact(last24hSummary?.tokens), failed: formatNumber(last24hSummary?.failures) }), <IconChartLine size={16} />)}
        {renderMetric(t('usage_statistics.failure_rate'), formatPercent(failedRate), `${formatNumber(failedRequests)} / ${formatNumber(usage.total_requests)}`, <IconShield size={16} />)}
        {renderMetric(t('usage_statistics.active_dimensions'), `${formatNumber(apiRows.length)} / ${formatNumber(modelRows.length)} / ${formatNumber(keyRows.length)}`, t('usage_statistics.active_dimensions_sub'), <IconModelCluster size={16} />)}
        {renderMetric(t('usage_statistics.last_request'), formatTimestamp(lastSeenMs), t('usage_statistics.active_period_sub', { days: formatNumber(activeDays), hours: formatNumber(activeHours) }), <IconTimer size={16} />)}
        {renderMetric(t('usage_statistics.peak_hour'), peakBucket ? `${peakBucket.label} · ${formatNumber(peakBucket.requests)}` : '-', peakBucket ? t('usage_statistics.recent_window_sub', { tokens: formatCompact(peakBucket.tokens), failed: formatNumber(peakBucket.failures) }) : t('usage_statistics.no_timeline'), <IconTrendingUp size={16} />)}
        {renderMetric(t('usage_statistics.top_model'), topModel, t('usage_statistics.by_requests'), <IconChartLine size={16} />)}
        {renderMetric(t('usage_statistics.top_key'), topKey, t('usage_statistics.by_requests'), <IconKey size={16} />)}
      </section>

      {!loading && !hasUsage && <div className={styles.emptyState}>{t('usage_statistics.empty')}</div>}

      <div className={styles.insightGrid}>
        <Card title={t('usage_statistics.recent_activity_title')}>
          <div className={styles.activityWindowGrid}>
            {activityWindowSummaries.map((summary) => (
              <div key={summary.key} className={styles.activityWindow}>
                <span className={styles.activityWindowLabel}>{t(`usage_statistics.activity_window_${summary.key}`)}</span>
                <strong>{formatNumber(summary.requests)}</strong>
                <span>{t('usage_statistics.recent_window_sub', { tokens: formatCompact(summary.tokens), failed: formatNumber(summary.failures) })}</span>
                <span>{t('usage_statistics.rate_window_sub', { rpm: formatRate(summary.rpm), tpm: formatRate(summary.tpm) })}</span>
              </div>
            ))}
          </div>
          <div className={styles.activityBars} aria-label={t('usage_statistics.recent_activity_title')}>
            {recentHourBuckets.map((bucket) => {
              const height = bucket.requests > 0 ? 16 + (bucket.requests / recentActivityMax) * 84 : 6;
              return (
                <div key={bucket.key} className={styles.activityBarSlot}>
                  <div
                    className={`${styles.activityBar} ${bucket.failures > 0 ? styles.activityBarWarning : ''}`}
                    title={t('usage_statistics.activity_hour_label', {
                      hour: bucket.key,
                      requests: formatNumber(bucket.requests),
                      tokens: formatCompact(bucket.tokens),
                      failed: formatNumber(bucket.failures),
                    })}
                  >
                    <span
                      className={styles.activityBarFill}
                      style={{ height: `${height}%` } as CSSProperties}
                    />
                  </div>
                  <span>{bucket.label}</span>
                </div>
              );
            })}
          </div>
        </Card>

        <Card title={t('usage_statistics.health_title')}>
          <div className={styles.healthSummary}>
            <div className={styles.healthRateBlock}>
              <span>{t('usage_statistics.success_rate')}</span>
              <strong>{formatPercent(100 - failedRate)}</strong>
              <small>{formatNumber(usage.success_count)} {t('usage_statistics.success')} / {formatNumber(failedRequests)} {t('usage_statistics.failed')}</small>
            </div>
            <div className={styles.healthTrack}>
              <span
                className={styles.healthSuccess}
                style={{
                  width: `${safeNumber(usage.total_requests) > 0 ? (safeNumber(usage.success_count) / safeNumber(usage.total_requests)) * 100 : 0}%`,
                }}
              />
              <span
                className={styles.healthFailure}
                style={{
                  width: `${safeNumber(usage.total_requests) > 0 ? (failedRequests / safeNumber(usage.total_requests)) * 100 : 0}%`,
                }}
              />
            </div>
          </div>
          <div className={styles.compactFacts}>
            <div>
              <span>{t('usage_statistics.details_captured')}</span>
              <strong>{formatNumber(detailRecords.length)}</strong>
            </div>
            <div>
              <span>{t('usage_statistics.active_days')}</span>
              <strong>{formatNumber(activeDays)}</strong>
            </div>
            <div>
              <span>{t('usage_statistics.active_hours')}</span>
              <strong>{formatNumber(activeHours)}</strong>
            </div>
            <div>
              <span>{t('usage_statistics.peak_hour')}</span>
              <strong>{peakBucket ? `${peakBucket.label} · ${formatNumber(peakBucket.requests)}` : '-'}</strong>
            </div>
          </div>
        </Card>
      </div>

      <div className={styles.insightGrid}>
        <Card title={t('usage_statistics.token_breakdown_title')}>
          <div className={styles.breakdownList}>
            {tokenBreakdownItems.map((item) => {
              const percentage = tokenBreakdownTotal > 0 ? (item.value / tokenBreakdownTotal) * 100 : 0;
              return (
                <div key={item.key} className={styles.breakdownRow}>
                  <div className={styles.breakdownLabelRow}>
                    <span className={styles.breakdownLabel}>
                      <span className={`${styles.breakdownDot} ${item.className}`} />
                      {item.label}
                    </span>
                    <strong>{formatNumber(item.value)}</strong>
                  </div>
                  <div className={styles.breakdownTrack}>
                    <span className={`${styles.breakdownFill} ${item.className}`} style={{ width: `${percentage}%` }} />
                  </div>
                  <span className={styles.breakdownPercent}>{percentage.toFixed(1)}%</span>
                </div>
              );
            })}
          </div>
        </Card>

        <Card title={t('usage_statistics.top_dimensions_title')}>
          <div className={styles.topLists}>
            {topDimensionGroups.map((group) => {
              const maxRequests = Math.max(...group.rows.map((row) => row.requests), 1);
              return (
                <div key={group.key} className={styles.topList}>
                  <h3>{group.title}</h3>
                  {group.rows.length ? (
                    group.rows.map((row) => (
                      <div key={`${group.key}-${row.id}`} className={styles.topListRow}>
                        <div className={styles.topListMeta}>
                          <span className={styles.monoCell}>{row.label}</span>
                          <small>
                            {formatNumber(row.requests)} {t('usage_statistics.requests')} · {formatCompact(row.tokens)} {t('usage_statistics.tokens')}
                          </small>
                        </div>
                        <div className={styles.topListBar}>
                          <span style={{ width: `${(row.requests / maxRequests) * 100}%` }} />
                        </div>
                      </div>
                    ))
                  ) : (
                    <div className={styles.emptyInline}>{t('usage_statistics.no_rows')}</div>
                  )}
                </div>
              );
            })}
          </div>
        </Card>
      </div>

      <Card
        title={t('usage_statistics.trend_title')}
        extra={
          <div className={styles.chartControls}>
            <div className={styles.segmentedControl}>
              {(['24h', '7d', '30d', 'all'] as TrendSpan[]).map((span) => (
                <button
                  type="button"
                  key={span}
                  className={trendSpan === span ? styles.segmentActive : ''}
                  onClick={() => setTrendSpan(span)}
                >
                  {t(`usage_statistics.span_${span}`)}
                </button>
              ))}
            </div>
            <div className={styles.segmentedControl}>
              {(['requests', 'tokens', 'cost'] as TrendMetric[]).map((metric) => (
                <button
                  type="button"
                  key={metric}
                  className={trendMetric === metric ? styles.segmentActive : ''}
                  onClick={() => setTrendMetric(metric)}
                >
                  {t(`usage_statistics.metric_${metric}`)}
                </button>
              ))}
            </div>
          </div>
        }
      >
        <TrendChart
          rows={trendRows}
          metric={trendMetric}
          formatValue={trendFormatter}
          emptyLabel={t('usage_statistics.no_timeline')}
        />
      </Card>

      <Card
        title={t('usage_statistics.breakdown_title')}
        extra={
          <div className={styles.segmentedControl}>
            {(['models', 'keys', 'apis', 'details'] as BreakdownTab[]).map((tab) => (
              <button
                type="button"
                key={tab}
                className={activeTab === tab ? styles.segmentActive : ''}
                onClick={() => setActiveTab(tab)}
              >
                {t(`usage_statistics.tab_${tab}`)}
              </button>
            ))}
          </div>
        }
      >
        {activeTab === 'models' && renderAggregateTable(modelRows, 'models')}
        {activeTab === 'keys' && renderAggregateTable(keyRows, 'keys')}
        {activeTab === 'apis' && renderAggregateTable(apiRows, 'apis')}
        {activeTab === 'details' && (
          <div className={styles.tableScroll}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>{t('usage_statistics.time')}</th>
                  <th>{t('usage_statistics.status')}</th>
                  <th>{t('usage_statistics.model')}</th>
                  <th>{t('usage_statistics.key')}</th>
                  <th>{t('usage_statistics.tokens')}</th>
                  <th>{t('usage_statistics.cached_tokens')}</th>
                  <th>{t('usage_statistics.estimated_cost')}</th>
                  <th>{t('usage_statistics.first_byte')}</th>
                  <th>{t('usage_statistics.latency')}</th>
                  <th>{t('usage_statistics.source')}</th>
                </tr>
              </thead>
              <tbody>
                {recentRows.length ? (
                  recentRows.map((row) => (
                    <tr key={row.id}>
                      <td>{formatTimestamp(row.timestampMs)}</td>
                      <td>
                        <span className={row.detail.failed ? styles.statusFailed : styles.statusOk}>
                          {row.detail.failed ? t('usage_statistics.failed') : t('usage_statistics.success')}
                        </span>
                      </td>
                      <td className={styles.monoCell}>{row.modelName}</td>
                      <td className={styles.monoCell}>{row.keyName}</td>
                      <td>{formatNumber(row.detail.tokens?.total_tokens)}</td>
                      <td>{formatNumber(row.detail.tokens?.cached_tokens)}</td>
                      <td>{formatCost(row.cost)}</td>
                      <td>{formatLatency(row.detail.first_byte_latency_ms)}</td>
                      <td>{formatLatency(row.detail.latency_ms)}</td>
                      <td className={styles.mutedCell}>{row.detail.source || row.detail.auth_index || row.apiName || '-'}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={10} className={styles.emptyCell}>
                      {t('usage_statistics.no_recent_rows')}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}
