import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  ArrowLeft,
  ArrowRight,
  Download,
  File,
  FileText,
  Folder,
  Grid2X2,
  Images,
  RefreshCw,
  Search,
  X,
  ZoomIn,
} from 'lucide-react';
import { getDesktopAuthToken, request } from '../../api';
import styles from './OutputsSurface.module.scss';

export type OutputItem = {
  id: string;
  path: string;
  name: string;
  kind: 'image' | 'document' | 'video' | 'audio' | 'text' | 'download';
  mime: string;
  size: number;
  modified: string;
  sha256?: string;
  origin: string;
  agent_id?: string;
  runtime_id?: string;
  revision?: number;
  revisions?: number;
};
type OutputRevision = { number: number; sha256: string; size: number; modified: string; snapshot: boolean };
type OutputList = {
  items: OutputItem[];
  total: number;
  has_more: boolean;
  scanned: number;
  truncated: boolean;
};
type FileEntry = {
  name: string;
  path: string;
  is_dir: boolean;
  kind?: string;
  size: number;
  modified: string;
};
type FileList = { path: string; entries: FileEntry[] };

const projectAPI = (projectId: string) => `/api/v1/projects/${encodeURIComponent(projectId)}`;
const assetPath = (projectId: string, path: string, mode: 'thumbnail' | 'view' | 'download') =>
  `${projectAPI(projectId)}/files?path=${encodeURIComponent(path)}&${mode}=1`;
const humanSize = (bytes: number) =>
  bytes < 1024
    ? `${bytes} B`
    : bytes < 1048576
      ? `${(bytes / 1024).toFixed(1)} KB`
      : `${(bytes / 1048576).toFixed(1)} MB`;

// Desktop clients use a bearer bootstrap token; remote/mobile clients use the
// Nexus HttpOnly session cookie. URLs and secrets never appear together.
function useAsset(url: string | null): string | null {
  const [src, setSrc] = useState<string | null>(null);
  useEffect(() => {
    if (!url) {
      setSrc(null);
      return;
    }
    const controller = new AbortController();
    let resource: string | null = null;
    setSrc(null);
    const token = getDesktopAuthToken();
    if (!token) {
      setSrc(url);
      return () => controller.abort();
    }
    void fetch(url, { signal: controller.signal, headers: { Authorization: `Bearer ${token}` } })
      .then(async (response) => {
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.blob();
      })
      .then((blob) => {
        if (controller.signal.aborted) return;
        resource = URL.createObjectURL(blob);
        setSrc(resource);
      })
      .catch(() => {
        if (!controller.signal.aborted) setSrc(null);
      });
    return () => {
      controller.abort();
      if (resource) URL.revokeObjectURL(resource);
    };
  }, [url]);
  return src;
}

const PreviewThumb: React.FC<{ projectId: string; item: OutputItem }> = ({ projectId, item }) => {
  const src = useAsset(item.kind === 'image' ? assetPath(projectId, item.path, 'thumbnail') : null);
  return src ? (
    <img src={src} alt="" loading="lazy" className={styles.thumbnail} />
  ) : (
    <div className={styles.placeholder}>
      {item.kind === 'image' ? <Images size={30} /> : <FileText size={30} />}
    </div>
  );
};

function download(projectId: string, rel: string, filename: string, version?: number) {
  const url = version
    ? `${projectAPI(projectId)}/files?path=${encodeURIComponent(rel)}&version=${version}`
    : assetPath(projectId, rel, 'download');
  const token = getDesktopAuthToken();
  if (!token) {
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    link.remove();
    return;
  }
  void fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(async (response) => {
      if (!response.ok) throw new Error(`Download failed: HTTP ${response.status}`);
      const blob = await response.blob();
      const objectUrl = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = objectUrl;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.setTimeout(() => URL.revokeObjectURL(objectUrl), 5000);
    })
    .catch((err: unknown) => window.alert(err instanceof Error ? err.message : String(err)));
}

const OutputPreview: React.FC<{ projectId: string; item: OutputItem; onClose: () => void }> = ({
  projectId,
  item,
  onClose,
}) => {
  const canPreview = item.kind !== 'download' && item.size <= 50 * 1024 * 1024;
  const src = useAsset(canPreview ? assetPath(projectId, item.path, 'view') : null);
  const [history, setHistory] = useState<OutputRevision[]>([]);
  useEffect(() => {
    if ((item.revisions || 0) <= 1) { setHistory([]); return; }
    let current = true;
    void request<{ versions: OutputRevision[] }>(`${projectAPI(projectId)}/files?path=${encodeURIComponent(item.path)}&history=1`)
      .then((data) => { if (current) setHistory(data.versions || []); })
      .catch(() => { if (current) setHistory([]); });
    return () => { current = false; };
  }, [projectId, item.path, item.revisions]);
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);
  return (
    <div
      className={styles.overlay}
      role="dialog"
      aria-modal="true"
      aria-label={`Prévia: ${item.name}`}
    >
      <div className={styles.preview}>
        <header className={styles.previewHeader}>
          <div>
            <strong>{item.name}</strong>
            <small>
              {humanSize(item.size)} · {item.origin} ·{' '}
              {item.modified ? new Date(item.modified).toLocaleString() : ''}
            </small>
          </div>
          <div className={styles.actions}>
            <button type="button" onClick={() => download(projectId, item.path, item.name)}>
              <Download size={16} /> Baixar original
            </button>
            <button type="button" onClick={onClose} aria-label="Fechar prévia">
              <X size={19} />
            </button>
          </div>
        </header>
        <div className={styles.viewer}>
          {src && item.kind === 'image' && (
            <img src={src} alt={item.name} className={styles.fullImage} />
          )}
          {src && item.kind === 'document' && (
            <iframe src={src} title={item.name} className={styles.fullDocument} />
          )}
          {src && item.kind === 'video' && (
            <video src={src} controls playsInline className={styles.fullImage} />
          )}
          {src && item.kind === 'audio' && <audio src={src} controls />}
          {src && item.kind === 'text' && (
            <iframe src={src} title={item.name} className={styles.fullDocument} sandbox="" />
          )}
          {!src && (
            <p>
              {canPreview
                ? 'Preparando prévia…'
                : 'Arquivo grande ou formato sem prévia. Baixe o original para abrir no seu dispositivo.'}
            </p>
          )}
        </div>
        {item.runtime_id && <div className={styles.provenance}>Gerado pelo runtime {item.runtime_id}{item.agent_id ? ` · agente ${item.agent_id}` : ''}</div>}
        {history.length > 0 && (
          <div className={styles.versions} aria-label="Histórico do arquivo">
            <strong>Versões anteriores</strong>
            {history.map((version) => (
              <button type="button" key={version.number}
                disabled={!version.snapshot}
                onClick={() => download(projectId, item.path, item.name, version.number)}>
                <Download size={14} /> v{version.number} · {humanSize(version.size)}
                {!version.snapshot && ' · original não armazenado'}
              </button>
            ))}
          </div>
        )}
        {item.sha256 && <footer className={styles.hash}>SHA-256: {item.sha256}</footer>
      </div>
    </div>
  );
};

export const OutputsSurface: React.FC<{ projectId: string }> = ({ projectId }) => {
  const [tab, setTab] = useState<'outputs' | 'files'>('outputs');
  const [kind, setKind] = useState('all');
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(0);
  const [listing, setListing] = useState<OutputList | null>(null);
  const [folder, setFolder] = useState('.');
  const [files, setFiles] = useState<FileEntry[]>([]);
  const [selected, setSelected] = useState<OutputItem | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const limit = 48;
  const refresh = useCallback(async () => {
    try {
      if (tab === 'outputs') {
        const result = await request<OutputList>(
          `${projectAPI(projectId)}/outputs?offset=${page * limit}&limit=${limit}&kind=${encodeURIComponent(kind)}`,
        );
        setListing(result);
      } else {
        const result = await request<FileList>(
          `${projectAPI(projectId)}/files?path=${encodeURIComponent(folder)}`,
        );
        setFiles(result.entries || []);
      }
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [projectId, tab, folder, page, kind]);
  useEffect(() => {
    setLoading(true);
    void refresh();
    const timer = window.setInterval(() => void refresh(), 8000);
    return () => window.clearInterval(timer);
  }, [refresh]);
  const items = useMemo(
    () =>
      (listing?.items || []).filter((item) =>
        item.name.toLowerCase().includes(query.toLowerCase()),
      ),
    [listing, query],
  );
  const displayedFiles = useMemo(
    () => files.filter((entry) => entry.name.toLowerCase().includes(query.toLowerCase())),
    [files, query],
  );
  const parentFolder = folder === '.' ? '.' : folder.split('/').slice(0, -1).join('/') || '.';
  const fileAsOutput = (entry: FileEntry): OutputItem => ({
    id: entry.path,
    path: entry.path,
    name: entry.name,
    kind: (entry.kind || 'download') as OutputItem['kind'],
    mime: '',
    size: entry.size,
    modified: entry.modified,
    origin: 'filesystem',
  });
  return (
    <section className={styles.root} aria-label="Resultados e arquivos do projeto">
      <header className={styles.header}>
        <div>
          <h2>Outputs</h2>
          <p>Resultados dos agentes e arquivos acessíveis no computador ou celular.</p>
        </div>
        <button type="button" onClick={() => void refresh()} aria-label="Atualizar resultados">
          <RefreshCw size={17} />
        </button>
      </header>
      <div className={styles.toolbar}>
        <div role="tablist" aria-label="Tipo de conteúdo" className={styles.tabs}>
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'outputs'}
            className={tab === 'outputs' ? styles.active : ''}
            onClick={() => {
              setTab('outputs');
              setPage(0);
            }}
          >
            <Grid2X2 size={16} /> Galeria
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'files'}
            className={tab === 'files' ? styles.active : ''}
            onClick={() => setTab('files')}
          >
            <Folder size={16} /> Arquivos
          </button>
        </div>
        <label className={styles.search}>
          <Search size={16} />
          <input
            aria-label="Filtrar pelo nome"
            placeholder="Filtrar resultados…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </label>
        {tab === 'outputs' && (
          <select
            aria-label="Tipo de arquivo"
            value={kind}
            onChange={(e) => {
              setKind(e.target.value);
              setPage(0);
            }}
          >
            <option value="all">Todos os formatos</option>
            <option value="image">Imagens</option>
            <option value="document">PDFs</option>
            <option value="video">Vídeos</option>
            <option value="audio">Áudios</option>
            <option value="text">Textos</option>
          </select>
        )}
      </div>
      {error && (
        <div className={styles.error} role="alert">
          {error}
        </div>
      )}
      {tab === 'outputs' ? (
        <>
          <div className={styles.summary}>
            {listing?.total ?? 0} resultados identificados
            {listing?.truncated ? ' · Limite de varredura atingido' : ''}
            {loading ? ' · Atualizando…' : ''}
          </div>
          <div className={styles.gallery}>
            {items.map((item) => (
              <button
                className={styles.tile}
                type="button"
                key={item.id}
                onClick={() => setSelected(item)}
                aria-label={`Visualizar ${item.name}`}
              >
                <PreviewThumb item={item} projectId={projectId} />
                <span className={styles.fileName} title={item.path}>
                  {item.name}
                </span>
                <small>
                  {item.kind} · {humanSize(item.size)}
                </small>
              </button>
            ))}
          </div>
          {items.length === 0 && !loading && (
            <p className={styles.empty}>
              Nenhum resultado encontrado. Arquivos PNG, JPG, PDF, vídeo ou texto criados no projeto
              aparecem aqui automaticamente.
            </p>
          )}
          <div className={styles.pagination}>
            <button
              type="button"
              onClick={() => setPage(Math.max(0, page - 1))}
              disabled={page === 0}
            >
              <ArrowLeft size={16} /> Anterior
            </button>
            <span>Página {page + 1}</span>
            <button type="button" onClick={() => setPage(page + 1)} disabled={!listing?.has_more}>
              Próxima <ArrowRight size={16} />
            </button>
          </div>
        </>
      ) : (
        <>
          <div className={styles.breadcrumb}>
            <button type="button" disabled={folder === '.'} onClick={() => setFolder(parentFolder)}>
              <ArrowLeft size={16} /> Voltar
            </button>
            <code>/{folder === '.' ? '' : folder}</code>
          </div>
          <div className={styles.fileRows} role="list" aria-label="Arquivos do projeto">
            {displayedFiles.map((entry) => (
              <div role="listitem" key={entry.path} className={styles.fileRow}>
                <button
                  type="button"
                  onClick={() =>
                    entry.is_dir ? setFolder(entry.path) : setSelected(fileAsOutput(entry))
                  }
                >
                  {entry.is_dir ? <Folder size={18} /> : <File size={18} />}
                  <span>{entry.name}</span>
                  {!entry.is_dir && <small>{humanSize(entry.size)}</small>}
                  {entry.is_dir && <ArrowRight size={15} />}
                </button>
                {!entry.is_dir && (
                  <button
                    type="button"
                    aria-label={`Baixar ${entry.name}`}
                    onClick={() => download(projectId, entry.path, entry.name)}
                  >
                    <Download size={16} />
                  </button>
                )}
              </div>
            ))}
          </div>
          {displayedFiles.length === 0 && !loading && (
            <p className={styles.empty}>
              Nenhum arquivo disponível nesta pasta. Arquivos ocultos, segredos e diretórios não
              autorizados não aparecem.
            </p>
          )}
        </>
      )}
      <div className={styles.notice}>
        <ZoomIn size={14} /> Originais permanecem no projeto. Pré-visualizações e links exigem
        autenticação no Nexus.
      </div>
      {selected && (
        <OutputPreview projectId={projectId} item={selected} onClose={() => setSelected(null)} />
      )}
    </section>
  );
};
