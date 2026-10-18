import type { ProviderInfo } from '../../types';

// Installation detection is a hint, NOT proof that a provider is authenticated
// or that any quota/billing is available. Registration is always opt-in.
export type FirstRunDetectionState = 'checking' | 'loaded' | 'unavailable';

export type FirstRunReadiness = {
  status: 'checking' | 'ready' | 'needs-account' | 'needs-provider' | 'unavailable';
  title: string;
  description: string;
  suggestedProvider?: string;
};

type DetectedProvider = Pick<ProviderInfo, 'id' | 'installed' | 'registration_state'>;

export const deriveFirstRunReadiness = (
  providers: DetectedProvider[],
  detectionState: FirstRunDetectionState,
): FirstRunReadiness => {
  if (detectionState === 'checking') {
    return {
      status: 'checking',
      title: 'Identificando seu ambiente…',
      description: 'Verificando provedores instalados e dependências sem alterar configurações.',
    };
  }
  if (detectionState === 'unavailable') {
    return {
      status: 'unavailable',
      title: 'Não foi possível verificar os provedores',
      description: 'Você ainda pode adicionar seu projeto. Tente verificar novamente quando o Nexus estiver disponível.',
    };
  }

  const installed = providers.filter((provider) => provider.installed);
  const authenticated = installed.filter(
    (provider) => provider.registration_state === 'REGISTERED_AUTHENTICATED',
  );
  if (authenticated.length > 0) {
    return {
      status: 'ready',
      title: authenticated.length === 1 ? 'Conta detectada' : `${authenticated.length} provedores com conta detectada`,
      description:
        'Já existe uma conta autenticada. Escolha a pasta do projeto e inicie sua sessão; a disponibilidade de quota será verificada separadamente.',
    };
  }
  if (installed.length > 0) {
    const next = installed.find(
      (provider) => provider.registration_state === 'PENDING_AUTH',
    ) ?? installed[0];
    return {
      status: 'needs-account',
      title: installed.length === 1 ? 'Provedor instalado' : `${installed.length} provedores instalados`,
      description:
        'O Nexus detectou a CLI, mas ainda não confirmou uma conta autenticada. Cadastre um perfil e conclua o login na seção Uso/Contas.',
      suggestedProvider: next.id,
    };
  }
  return {
    status: 'needs-provider',
    title: 'Nenhum provedor de IA detectado',
    description:
      'Instale Codex, Claude ou outra CLI compatível a partir da fonte oficial. Você já pode cadastrar o projeto enquanto isso.',
  };
};
