import { useEffect, useState } from "react";
import { GetConfig } from "../../wailsjs/go/gui/App";
import type { Config } from "../types";

interface ConfigPanelProps {
  onClose: () => void;
}

const CARD_DEFAULT =
  "border-blue-400 dark:border-blue-500 bg-blue-50 dark:bg-blue-900/20";
const CARD_NORMAL = "border-gray-200 dark:border-gray-700";

function DefaultBadge() {
  return (
    <span className="text-xs px-2 py-0.5 rounded bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300 font-semibold">
      デフォルト
    </span>
  );
}

export function ConfigPanel({ onClose }: ConfigPanelProps) {
  const [config, setConfig] = useState<Config | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    GetConfig()
      .then((cfg) => {
        if (!cancelled) setConfig(cfg);
      })
      .catch((err) => {
        if (!cancelled)
          setError(err instanceof Error ? err.message : String(err));
      });
    return () => {
      cancelled = true;
    };
  }, []);

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

      {error && <ConfigError message={error} />}
      {!error && !config && (
        <p className="text-gray-400 dark:text-gray-500 animate-pulse">
          読み込み中…
        </p>
      )}
      {config && <ConfigView config={config} />}
    </div>
  );
}

function ConfigError({ message }: { message: string }) {
  return (
    <div className="p-4 rounded bg-yellow-50 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200">
      <p className="font-semibold mb-2">設定ファイルを読み込めませんでした</p>
      <p className="text-sm mb-3">{message}</p>
      <p className="text-sm">
        以下のいずれかの場所に設定ファイルを配置してください:
      </p>
      <ul className="text-sm mt-1 list-disc list-inside">
        <li>
          <code className="bg-yellow-100 dark:bg-yellow-900 px-1 rounded">
            ./yomite.json
          </code>
          （カレントディレクトリ）
        </li>
        <li>
          <code className="bg-yellow-100 dark:bg-yellow-900 px-1 rounded">
            ~/.config/yomite/config.json
          </code>
          （グローバル設定）
        </li>
      </ul>
    </div>
  );
}

function ConfigView({ config }: { config: Config }) {
  return (
    <div className="flex-1 overflow-y-auto space-y-6">
      <ProvidersSection
        providers={config.providers}
        defaultProvider={config.default_provider}
      />
      <PersonasSection
        personas={config.personas}
        defaultPersona={config.default_persona}
      />
    </div>
  );
}

function ProvidersSection({
  providers,
  defaultProvider,
}: {
  providers: Config["providers"];
  defaultProvider: string;
}) {
  const entries = Object.entries(providers);
  return (
    <section>
      <h3 className="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-2">
        プロバイダ一覧
      </h3>
      {entries.length === 0 ? (
        <p className="text-sm text-gray-400">プロバイダが設定されていません</p>
      ) : (
        <div className="space-y-2">
          {entries.map(([id, provider]) => (
            <ProviderCard
              key={id}
              id={id}
              provider={provider}
              isDefault={id === defaultProvider}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function PersonasSection({
  personas,
  defaultPersona,
}: {
  personas: Config["personas"];
  defaultPersona: string;
}) {
  const entries = Object.entries(personas);
  return (
    <section>
      <h3 className="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-2">
        ペルソナ一覧
      </h3>
      {entries.length === 0 ? (
        <p className="text-sm text-gray-400">ペルソナが設定されていません</p>
      ) : (
        <div className="space-y-2">
          {entries.map(([id, persona]) => (
            <PersonaCard
              key={id}
              id={id}
              persona={persona}
              isDefault={id === defaultPersona}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function ProviderCard({
  id,
  provider,
  isDefault,
}: {
  id: string;
  provider: Config["providers"][string];
  isDefault: boolean;
}) {
  return (
    <div
      className={`p-3 rounded border ${isDefault ? CARD_DEFAULT : CARD_NORMAL}`}
    >
      <div className="flex items-center gap-2 mb-1">
        <span className="font-medium text-sm">{id}</span>
        {isDefault && <DefaultBadge />}
      </div>
      <div className="grid grid-cols-3 gap-2 text-xs text-gray-600 dark:text-gray-400">
        <div>
          <span className="text-gray-400 dark:text-gray-500">type:</span>{" "}
          {provider.type}
        </div>
        <div>
          <span className="text-gray-400 dark:text-gray-500">model:</span>{" "}
          {provider.model}
        </div>
        <div>
          <span className="text-gray-400 dark:text-gray-500">origin:</span>{" "}
          {provider.origin}
        </div>
      </div>
    </div>
  );
}

function PersonaCard({
  id,
  persona,
  isDefault,
}: {
  id: string;
  persona: Config["personas"][string];
  isDefault: boolean;
}) {
  const [promptOpen, setPromptOpen] = useState(false);

  return (
    <div
      className={`p-3 rounded border ${isDefault ? CARD_DEFAULT : CARD_NORMAL}`}
    >
      <div className="flex items-center gap-2 mb-1">
        <span className="font-medium text-sm">{persona.display_name}</span>
        <span className="text-xs text-gray-400">({id})</span>
        {isDefault && <DefaultBadge />}
      </div>
      {/* NOTE: 設定ビューアはJSON設定ファイルのフィールド名をそのまま表示する。
          ユーザーが設定ファイルを編集する際の対応関係を明確にするため。 */}
      <div className="flex gap-4 text-xs text-gray-600 dark:text-gray-400 mb-2">
        <div>
          <span className="text-gray-400 dark:text-gray-500">
            memory_capacity:
          </span>{" "}
          {persona.memory_capacity}
        </div>
        <div>
          <span className="text-gray-400 dark:text-gray-500">max_steps:</span>{" "}
          {persona.max_steps}
        </div>
      </div>
      {persona.system_prompt && (
        <div>
          <button
            onClick={() => setPromptOpen((prev) => !prev)}
            className="text-xs text-blue-600 dark:text-blue-400 hover:underline"
          >
            {promptOpen ? "▼ system_prompt を閉じる" : "▶ system_prompt を表示"}
          </button>
          {promptOpen && (
            <pre className="mt-1 p-2 text-xs bg-gray-100 dark:bg-gray-800 rounded whitespace-pre-wrap break-words max-h-48 overflow-y-auto">
              {persona.system_prompt}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}
