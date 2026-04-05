import { useCallback, useEffect, useState } from "react";
import {
  GetConfigPath,
  GetDefaultConfigPath,
  LoadConfigFromPath,
  SaveConfig,
} from "../../wailsjs/go/gui/App";
import { core } from "../../wailsjs/go/models";
import type { Config, Persona, ProviderConfig } from "../types";

interface ConfigPanelProps {
  onClose: () => void;
}

const LOG_LEVELS = ["debug", "info", "warn"] as const;

const INPUT_CLASS =
  "mt-0.5 w-full px-2 py-1.5 text-sm border rounded border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800";
const INPUT_SMALL_CLASS =
  "mt-0.5 w-full px-2 py-1 text-xs border rounded border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800";

function emptyConfig(): Config {
  return {
    log: { level: "info", path: "" },
    default_provider: "",
    default_persona: "",
    providers: {},
    personas: {},
  };
}

export function ConfigPanel({ onClose }: ConfigPanelProps) {
  const [configPath, setConfigPath] = useState("");
  const [config, setConfig] = useState<Config | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saveMessage, setSaveMessage] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const explicitPath = await GetConfigPath();
        const path =
          explicitPath || (await GetDefaultConfigPath());
        if (cancelled) return;
        setConfigPath(path);

        const cfg = await LoadConfigFromPath(path);
        if (!cancelled) setConfig(cfg);
      } catch (err) {
        if (!cancelled) {
          // NOTE: 設定ファイルが存在しない場合は空フォームにフォールバックする。
          // 初回起動時や新規作成ユースケースを想定している。
          setConfig(emptyConfig());
          setError(
            err instanceof Error ? err.message : String(err),
          );
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const handleLoad = useCallback(async () => {
    setLoading(true);
    setError(null);
    setSaveMessage(null);
    try {
      const cfg = await LoadConfigFromPath(configPath);
      setConfig(cfg);
    } catch {
      // NOTE: ファイル未存在時は空フォームで新規作成を可能にする。
      setConfig(emptyConfig());
      setError("ファイルが見つかりません。新規作成用の空フォームを表示します。");
    } finally {
      setLoading(false);
    }
  }, [configPath]);

  const handleSave = useCallback(async () => {
    if (!config) return;
    setSaveMessage(null);
    setError(null);
    try {
      await SaveConfig(configPath, core.Config.createFrom(config));
      setSaveMessage("保存しました");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [configPath, config]);

  const updateConfig = useCallback(
    (updater: (prev: Config) => Config) => {
      setConfig((prev) => (prev ? updater(prev) : prev));
      setSaveMessage(null);
    },
    [],
  );

  return (
    <div className="flex-1 flex flex-col min-h-0 p-4">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-bold">設定</h2>
        <button
          onClick={onClose}
          className="px-3 py-1 text-sm rounded border border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
        >
          閉じる
        </button>
      </div>

      {/* パス入力 */}
      <div className="flex gap-2 mb-4">
        <input
          type="text"
          value={configPath}
          onChange={(e) => setConfigPath(e.target.value)}
          placeholder="設定ファイルのパス"
          className="flex-1 px-3 py-1.5 text-sm border rounded border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800"
        />
        <button
          onClick={handleLoad}
          className="px-3 py-1.5 text-sm rounded bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
        >
          読み込み
        </button>
      </div>

      {error && (
        <div className="mb-3 p-3 rounded bg-yellow-50 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 text-sm">
          {error}
        </div>
      )}
      {saveMessage && (
        <div className="mb-3 p-3 rounded bg-green-50 dark:bg-green-900/30 text-green-800 dark:text-green-200 text-sm">
          {saveMessage}
        </div>
      )}

      {loading && (
        <p className="text-gray-400 dark:text-gray-500 animate-pulse">
          読み込み中…
        </p>
      )}

      {!loading && config && (
        <div className="flex-1 overflow-y-auto space-y-6">
          <LogSection
            log={config.log}
            onChange={(log) => updateConfig((c) => ({ ...c, log }))}
          />
          <ProvidersSection
            providers={config.providers}
            defaultProvider={config.default_provider}
            onChangeProviders={(providers) =>
              updateConfig((c) => ({ ...c, providers }))
            }
            onChangeDefault={(default_provider) =>
              updateConfig((c) => ({ ...c, default_provider }))
            }
          />
          <PersonasSection
            personas={config.personas}
            defaultPersona={config.default_persona}
            onChangePersonas={(personas) =>
              updateConfig((c) => ({ ...c, personas }))
            }
            onChangeDefault={(default_persona) =>
              updateConfig((c) => ({ ...c, default_persona }))
            }
          />

          <div className="pb-4">
            <button
              onClick={handleSave}
              className="px-6 py-2 text-sm font-semibold rounded bg-blue-600 text-white hover:bg-blue-700 transition-colors"
            >
              保存
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

/* ---------- ログ設定 ---------- */

function LogSection({
  log,
  onChange,
}: {
  log: Config["log"];
  onChange: (log: Config["log"]) => void;
}) {
  return (
    <section>
      <h3 className="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-2">
        ログ設定
      </h3>
      <div className="grid grid-cols-2 gap-3">
        <label className="block">
          <span className="text-xs text-gray-500 dark:text-gray-400">
            level
          </span>
          <select
            value={log.level}
            onChange={(e) => onChange({ ...log, level: e.target.value })}
            className={INPUT_CLASS}
          >
            {LOG_LEVELS.map((l) => (
              <option key={l} value={l}>
                {l}
              </option>
            ))}
          </select>
        </label>
        <label className="block">
          <span className="text-xs text-gray-500 dark:text-gray-400">
            path
          </span>
          <input
            type="text"
            value={log.path}
            onChange={(e) => onChange({ ...log, path: e.target.value })}
            className={INPUT_CLASS}
          />
        </label>
      </div>
    </section>
  );
}

/* ---------- プロバイダ ---------- */

function ProvidersSection({
  providers,
  defaultProvider,
  onChangeProviders,
  onChangeDefault,
}: {
  providers: Config["providers"];
  defaultProvider: string;
  onChangeProviders: (providers: Config["providers"]) => void;
  onChangeDefault: (id: string) => void;
}) {
  const ids = Object.keys(providers);

  function handleAdd() {
    const id = prompt("プロバイダIDを入力してください");
    if (!id || id in providers) return;
    onChangeProviders({
      ...providers,
      [id]: { type: "ollama", model: "", origin: "http://localhost:11434" },
    });
  }

  function handleDelete(id: string) {
    const next = { ...providers };
    delete next[id];
    onChangeProviders(next);
    if (defaultProvider === id) onChangeDefault("");
  }

  function handleUpdate(id: string, provider: ProviderConfig) {
    onChangeProviders({ ...providers, [id]: provider });
  }

  return (
    <section>
      <h3 className="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-2">
        プロバイダ
      </h3>

      {ids.length > 0 && (
        <label className="block mb-3">
          <span className="text-xs text-gray-500 dark:text-gray-400">
            デフォルト
          </span>
          <select
            value={defaultProvider}
            onChange={(e) => onChangeDefault(e.target.value)}
            className={INPUT_CLASS}
          >
            <option value="">（未選択）</option>
            {ids.map((id) => (
              <option key={id} value={id}>
                {id}
              </option>
            ))}
          </select>
        </label>
      )}

      <div className="space-y-2">
        {ids.map((id) => (
          <ProviderCard
            key={id}
            id={id}
            provider={providers[id]}
            onChange={(p) => handleUpdate(id, p)}
            onDelete={() => handleDelete(id)}
          />
        ))}
      </div>

      <button
        onClick={handleAdd}
        className="mt-2 px-3 py-1 text-xs rounded border border-dashed border-gray-400 dark:border-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
      >
        ＋ 追加
      </button>
    </section>
  );
}

function ProviderCard({
  id,
  provider,
  onChange,
  onDelete,
}: {
  id: string;
  provider: ProviderConfig;
  onChange: (p: ProviderConfig) => void;
  onDelete: () => void;
}) {
  return (
    <div className="p-3 rounded border border-gray-200 dark:border-gray-700">
      <div className="flex items-center justify-between mb-2">
        <span className="font-medium text-sm">{id}</span>
        <button
          onClick={onDelete}
          className="text-xs text-red-500 hover:text-red-700 dark:hover:text-red-400"
        >
          削除
        </button>
      </div>
      <div className="grid grid-cols-3 gap-2">
        <label className="block">
          <span className="text-xs text-gray-500 dark:text-gray-400">type</span>
          <input
            type="text"
            value={provider.type}
            onChange={(e) => onChange({ ...provider, type: e.target.value })}
            className={INPUT_SMALL_CLASS}
          />
        </label>
        <label className="block">
          <span className="text-xs text-gray-500 dark:text-gray-400">
            model
          </span>
          <input
            type="text"
            value={provider.model}
            onChange={(e) => onChange({ ...provider, model: e.target.value })}
            className={INPUT_SMALL_CLASS}
          />
        </label>
        <label className="block">
          <span className="text-xs text-gray-500 dark:text-gray-400">
            origin
          </span>
          <input
            type="text"
            value={provider.origin}
            onChange={(e) => onChange({ ...provider, origin: e.target.value })}
            className={INPUT_SMALL_CLASS}
          />
        </label>
      </div>
    </div>
  );
}

/* ---------- ペルソナ ---------- */

function PersonasSection({
  personas,
  defaultPersona,
  onChangePersonas,
  onChangeDefault,
}: {
  personas: Config["personas"];
  defaultPersona: string;
  onChangePersonas: (personas: Config["personas"]) => void;
  onChangeDefault: (id: string) => void;
}) {
  const ids = Object.keys(personas);

  function handleAdd() {
    const id = prompt("ペルソナIDを入力してください");
    if (!id || id in personas) return;
    onChangePersonas({
      ...personas,
      [id]: {
        display_name: "",
        system_prompt: "",
        memory_capacity: 500,
        max_steps: 60,
      },
    });
  }

  function handleDelete(id: string) {
    const next = { ...personas };
    delete next[id];
    onChangePersonas(next);
    if (defaultPersona === id) onChangeDefault("");
  }

  function handleUpdate(id: string, persona: Persona) {
    onChangePersonas({ ...personas, [id]: persona });
  }

  return (
    <section>
      <h3 className="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-2">
        ペルソナ
      </h3>

      {ids.length > 0 && (
        <label className="block mb-3">
          <span className="text-xs text-gray-500 dark:text-gray-400">
            デフォルト
          </span>
          <select
            value={defaultPersona}
            onChange={(e) => onChangeDefault(e.target.value)}
            className={INPUT_CLASS}
          >
            <option value="">（未選択）</option>
            {ids.map((id) => (
              <option key={id} value={id}>
                {id}
              </option>
            ))}
          </select>
        </label>
      )}

      <div className="space-y-2">
        {ids.map((id) => (
          <PersonaCard
            key={id}
            id={id}
            persona={personas[id]}
            onChange={(p) => handleUpdate(id, p)}
            onDelete={() => handleDelete(id)}
          />
        ))}
      </div>

      <button
        onClick={handleAdd}
        className="mt-2 px-3 py-1 text-xs rounded border border-dashed border-gray-400 dark:border-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
      >
        ＋ 追加
      </button>
    </section>
  );
}

function PersonaCard({
  id,
  persona,
  onChange,
  onDelete,
}: {
  id: string;
  persona: Persona;
  onChange: (p: Persona) => void;
  onDelete: () => void;
}) {
  return (
    <div className="p-3 rounded border border-gray-200 dark:border-gray-700">
      <div className="flex items-center justify-between mb-2">
        <span className="font-medium text-sm">{id}</span>
        <button
          onClick={onDelete}
          className="text-xs text-red-500 hover:text-red-700 dark:hover:text-red-400"
        >
          削除
        </button>
      </div>
      <div className="grid grid-cols-3 gap-2 mb-2">
        <label className="block">
          <span className="text-xs text-gray-500 dark:text-gray-400">
            display_name
          </span>
          <input
            type="text"
            value={persona.display_name}
            onChange={(e) =>
              onChange({ ...persona, display_name: e.target.value })
            }
            className={INPUT_SMALL_CLASS}
          />
        </label>
        <label className="block">
          <span className="text-xs text-gray-500 dark:text-gray-400">
            memory_capacity
          </span>
          <input
            type="number"
            value={persona.memory_capacity}
            onChange={(e) =>
              onChange({
                ...persona,
                memory_capacity: parseInt(e.target.value, 10) || 0,
              })
            }
            className={INPUT_SMALL_CLASS}
          />
        </label>
        <label className="block">
          <span className="text-xs text-gray-500 dark:text-gray-400">
            max_steps
          </span>
          <input
            type="number"
            value={persona.max_steps}
            onChange={(e) =>
              onChange({
                ...persona,
                max_steps: parseInt(e.target.value, 10) || 0,
              })
            }
            className={INPUT_SMALL_CLASS}
          />
        </label>
      </div>
      <label className="block">
        <span className="text-xs text-gray-500 dark:text-gray-400">
          system_prompt
        </span>
        <textarea
          value={persona.system_prompt}
          onChange={(e) =>
            onChange({ ...persona, system_prompt: e.target.value })
          }
          rows={3}
          className={`${INPUT_SMALL_CLASS} resize-y`}
        />
      </label>
    </div>
  );
}
