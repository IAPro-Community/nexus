export const providerLoginCommand = (provider: string, profile: string): string =>
  profile === provider ? `nexus login ${provider}` : `nexus login ${provider} ${profile}`;
