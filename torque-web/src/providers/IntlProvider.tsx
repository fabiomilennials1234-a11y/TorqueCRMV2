import { IntlProvider as ReactIntlProvider } from "react-intl";
import type { ReactNode } from "react";
import messages from "@/i18n/pt-BR";

interface IntlProviderProps {
  children: ReactNode;
}

export function IntlProvider({ children }: IntlProviderProps) {
  return (
    <ReactIntlProvider
      locale="pt-BR"
      messages={messages}
      defaultLocale="pt-BR"
      onError={() => {
        // Suppress missing-translation warnings in dev
      }}
    >
      {children}
    </ReactIntlProvider>
  );
}
