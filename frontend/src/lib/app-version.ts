export type AppVersionManifest = {
  version: string;
  builtAt: string;
};

export const APP_VERSION = __APP_VERSION__;
export const APP_BUILD_TIME = __APP_BUILD_TIME__;
export const APP_VERSION_PARAM = "appv";

export async function fetchLatestAppVersion() {
  const response = await fetch(versionManifestURL(), {
    cache: "no-store",
    headers: {
      Pragma: "no-cache",
      "Cache-Control": "no-cache",
    },
  });

  if (!response.ok) {
    throw new Error(`Unable to fetch version manifest: ${response.status}`);
  }

  return (await response.json()) as AppVersionManifest;
}

export function buildRefreshURL(version: string, currentHref = window.location.href) {
  const url = new URL(currentHref);
  url.searchParams.set(APP_VERSION_PARAM, version);
  return url.toString();
}

export function refreshToLatestVersion(version: string, locationObject = window.location) {
  locationObject.replace(buildRefreshURL(version, locationObject.href));
}

export function clearRefreshVersionParam(currentHref = window.location.href) {
  const url = new URL(currentHref);
  if (!url.searchParams.has(APP_VERSION_PARAM)) {
    return;
  }

  url.searchParams.delete(APP_VERSION_PARAM);
  window.history.replaceState(window.history.state, "", `${url.pathname}${url.search}${url.hash}`);
}

function versionManifestURL(now = Date.now()) {
  const url = new URL("/version.json", window.location.origin);
  url.searchParams.set("ts", String(now));
  return url.toString();
}
