export type PayloadParamValueType = 'string' | 'number' | 'boolean' | 'json';
export type PayloadParamValidationErrorCode =
  | 'payload_invalid_number'
  | 'payload_invalid_boolean'
  | 'payload_invalid_json';

export type VisualConfigFieldPath =
  | 'port'
  | 'apiKeyControlsYaml'
  | 'pprofAddr'
  | 'logsMaxTotalSizeMb'
  | 'errorLogsMaxFiles'
  | 'usageStatisticsFlushIntervalSeconds'
  | 'conversationLogMaxFileSizeMb'
  | 'conversationLogMaxTotalSizeMb'
  | 'conversationLogMaxEntryBytes'
  | 'presetPromptMaxBytes'
  | 'redisUsageQueueRetentionSeconds'
  | 'requestRetry'
  | 'maxRetryCredentials'
  | 'maxRetryInterval'
  | 'upstreamConcurrencyDefault'
  | 'upstreamConcurrencyProvidersYaml'
  | 'upstreamConcurrencyQueueTimeoutSeconds'
  | 'streaming.keepaliveSeconds'
  | 'streaming.bootstrapRetries'
  | 'streaming.nonstreamKeepaliveInterval';

export type VisualConfigValidationErrorCode =
  | 'port_range'
  | 'non_negative_integer'
  | 'invalid_yaml'
  | 'yaml_array'
  | 'yaml_map';

export type VisualConfigValidationErrors = Partial<
  Record<VisualConfigFieldPath, VisualConfigValidationErrorCode>
>;

export type PayloadParamEntry = {
  id: string;
  path: string;
  valueType: PayloadParamValueType;
  value: string;
};

export type PayloadModelEntry = {
  id: string;
  name: string;
  protocol?: string;
};

export type PayloadRule = {
  id: string;
  models: PayloadModelEntry[];
  params: PayloadParamEntry[];
};

export type PayloadFilterRule = {
  id: string;
  models: PayloadModelEntry[];
  params: string[];
};

export interface StreamingConfig {
  keepaliveSeconds: string;
  bootstrapRetries: string;
  nonstreamKeepaliveInterval: string;
}

export type DisableImageGenerationModeValue = 'false' | 'true' | 'chat';

export type VisualConfigValues = {
  host: string;
  port: string;
  tlsEnable: boolean;
  tlsCert: string;
  tlsKey: string;
  rmAllowRemote: boolean;
  rmSecretKey: string;
  rmDisableControlPanel: boolean;
  rmDisableAutoUpdatePanel: boolean;
  rmPanelRepo: string;
  authDir: string;
  apiKeysText: string;
  apiKeyControlsYaml: string;
  debug: boolean;
  pprofEnable: boolean;
  pprofAddr: string;
  commercialMode: boolean;
  loggingToFile: boolean;
  logsMaxTotalSizeMb: string;
  errorLogsMaxFiles: string;
  usageStatisticsEnabled: boolean;
  usageStatisticsPath: string;
  usageStatisticsFlushIntervalSeconds: string;
  conversationLogEnabled: boolean;
  conversationLogDirectory: string;
  conversationLogMaxFileSizeMb: string;
  conversationLogMaxTotalSizeMb: string;
  conversationLogMaxEntryBytes: string;
  presetPromptEnabled: boolean;
  presetPromptPrompt: string;
  presetPromptMaxBytes: string;
  redisUsageQueueRetentionSeconds: string;
  proxyUrl: string;
  forceModelPrefix: boolean;
  passthroughHeaders: boolean;
  requestRetry: string;
  maxRetryCredentials: string;
  maxRetryInterval: string;
  disableCooling: boolean;
  upstreamConcurrencyDefault: string;
  upstreamConcurrencyProvidersYaml: string;
  upstreamConcurrencyQueueTimeoutSeconds: string;
  disableImageGeneration: DisableImageGenerationModeValue;
  quotaSwitchProject: boolean;
  quotaSwitchPreviewModel: boolean;
  quotaAntigravityCredits: boolean;
  routingStrategy: 'round-robin' | 'fill-first';
  routingSessionAffinity: boolean;
  routingSessionAffinityTTL: string;
  wsAuth: boolean;
  payloadDefaultRules: PayloadRule[];
  payloadDefaultRawRules: PayloadRule[];
  payloadOverrideRules: PayloadRule[];
  payloadOverrideRawRules: PayloadRule[];
  payloadFilterRules: PayloadFilterRule[];
  streaming: StreamingConfig;
};

export const makeClientId = () => {
  if (typeof globalThis.crypto?.randomUUID === 'function') return globalThis.crypto.randomUUID();
  return `${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`;
};

export const DEFAULT_VISUAL_VALUES: VisualConfigValues = {
  host: '',
  port: '',
  tlsEnable: false,
  tlsCert: '',
  tlsKey: '',
  rmAllowRemote: false,
  rmSecretKey: '',
  rmDisableControlPanel: false,
  rmDisableAutoUpdatePanel: false,
  rmPanelRepo: '',
  authDir: '',
  apiKeysText: '',
  apiKeyControlsYaml: '',
  debug: false,
  pprofEnable: false,
  pprofAddr: '',
  commercialMode: false,
  loggingToFile: false,
  logsMaxTotalSizeMb: '',
  errorLogsMaxFiles: '',
  usageStatisticsEnabled: false,
  usageStatisticsPath: '',
  usageStatisticsFlushIntervalSeconds: '',
  conversationLogEnabled: false,
  conversationLogDirectory: '',
  conversationLogMaxFileSizeMb: '',
  conversationLogMaxTotalSizeMb: '',
  conversationLogMaxEntryBytes: '',
  presetPromptEnabled: false,
  presetPromptPrompt: '',
  presetPromptMaxBytes: '',
  redisUsageQueueRetentionSeconds: '',
  proxyUrl: '',
  forceModelPrefix: false,
  passthroughHeaders: false,
  requestRetry: '',
  maxRetryCredentials: '',
  maxRetryInterval: '',
  disableCooling: false,
  upstreamConcurrencyDefault: '',
  upstreamConcurrencyProvidersYaml: '',
  upstreamConcurrencyQueueTimeoutSeconds: '',
  disableImageGeneration: 'false',
  quotaSwitchProject: true,
  quotaSwitchPreviewModel: true,
  quotaAntigravityCredits: false,
  routingStrategy: 'round-robin',
  routingSessionAffinity: false,
  routingSessionAffinityTTL: '',
  wsAuth: false,
  payloadDefaultRules: [],
  payloadDefaultRawRules: [],
  payloadOverrideRules: [],
  payloadOverrideRawRules: [],
  payloadFilterRules: [],
  streaming: {
    keepaliveSeconds: '',
    bootstrapRetries: '',
    nonstreamKeepaliveInterval: '',
  },
};
