import { useMemo } from "react";
import { toast } from "sonner";
import { useI18n } from "./i18n";

type NotificationAction = {
  label: string;
  onClick: () => void;
};

type MessageResolver<T> = string | ((value: T) => string);
type ErrorResolver = string | ((error: unknown) => string);

type RequestToastOptions<T> = {
  loadingKey: string;
  successKey: MessageResolver<T>;
  errorKey: ErrorResolver;
  action?: (value: T) => NotificationAction | undefined;
};

export function useNotifications() {
  const { t } = useI18n();

  return useMemo(() => {
    async function track<T>(request: Promise<T>, options: RequestToastOptions<T>) {
      const toastId = toast.loading(t(options.loadingKey));

      try {
        const value = await request;
        const action = options.action?.(value);

        toast.success(resolveMessage(value, options.successKey, t), {
          id: toastId,
          ...(action ? { action } : {}),
        });

        return value;
      } catch (error) {
        toast.error(resolveError(error, options.errorKey, t), { id: toastId });
        throw error;
      }
    }

    return {
      track,
      login<T>(request: Promise<T>) {
        return track(request, {
          loadingKey: "toast.signingIn",
          successKey: "toast.loginSuccess",
          errorKey: "toast.loginFailed",
        });
      },
      assignment<T>(request: Promise<T>) {
        return track(request, {
          loadingKey: "toast.assignmentSaving",
          successKey: "toast.assignmentSaved",
          errorKey: "toast.assignmentFailed",
        });
      },
      reply<T>(request: Promise<T>) {
        return track(request, {
          loadingKey: "toast.replySending",
          successKey: "toast.replySent",
          errorKey: "toast.replyFailed",
        });
      },
      ticketCreation<T>(
        request: Promise<T>,
        options?: { action?: (value: T) => NotificationAction | undefined },
      ) {
        return track(request, {
          loadingKey: "toast.ticketCreating",
          successKey: "toast.ticketCreated",
          errorKey: "toast.ticketCreateFailed",
          action: options?.action,
        });
      },
      ticketStatus<T>(request: Promise<T>) {
        return track(request, {
          loadingKey: "toast.ticketStatusSaving",
          successKey: "toast.ticketStatusSaved",
          errorKey: "toast.ticketStatusFailed",
        });
      },
      errorKey(key: string) {
        toast.error(t(key));
      },
      languageChanged(nextLocale: string) {
        const language = nextLocale === "en" ? "English" : "Bahasa Indonesia";
        toast.success(t("toast.languageChanged", { language }));
      },
      logoutSuccess() {
        toast.success(t("toast.logoutSuccess"));
      },
      workspaceRestored() {
        toast.success(t("toast.workspaceRestored"));
      },
      themeChanged(nextMode: "light" | "dark") {
        toast.success(nextMode === "dark" ? t("toast.themeDark") : t("toast.themeLight"));
      },
    };
  }, [t]);
}

function resolveMessage<T>(value: T, resolver: MessageResolver<T>, t: (key: string) => string) {
  return typeof resolver === "function" ? resolver(value) : t(resolver);
}

function resolveError(error: unknown, resolver: ErrorResolver, t: (key: string) => string) {
  return typeof resolver === "function" ? resolver(error) : t(resolver);
}
