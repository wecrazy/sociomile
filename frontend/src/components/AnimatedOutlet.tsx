import { useEffect, useState } from "react";
import { useLocation, useOutlet } from "react-router-dom";

const ROUTE_TRANSITION_MS = 220;

type TransitionStage = "enter" | "exit" | "idle";

export function AnimatedOutlet() {
  const location = useLocation();
  const outlet = useOutlet();
  const [displayOutlet, setDisplayOutlet] = useState(outlet);
  const [displayPathname, setDisplayPathname] = useState(location.pathname);
  const [stage, setStage] = useState<TransitionStage>("enter");

  useEffect(() => {
    if (location.pathname === displayPathname) {
      return;
    }

    if (prefersReducedMotion()) {
      setDisplayOutlet(outlet);
      setDisplayPathname(location.pathname);
      setStage("idle");
      return;
    }

    setStage("exit");

    const timeoutId = window.setTimeout(() => {
      setDisplayOutlet(outlet);
      setDisplayPathname(location.pathname);
      setStage("enter");
    }, ROUTE_TRANSITION_MS);

    return () => window.clearTimeout(timeoutId);
  }, [displayPathname, location.pathname, outlet]);

  useEffect(() => {
    if (stage !== "enter") {
      return;
    }

    const timeoutId = window.setTimeout(() => {
      setStage("idle");
    }, ROUTE_TRANSITION_MS + 40);

    return () => window.clearTimeout(timeoutId);
  }, [displayPathname, stage]);

  return (
    <div className="route-stage">
      <div
        className={`route-transition-shell is-${stage}`}
        data-route-kind={getRouteKind(displayPathname)}
      >
        {displayOutlet}
      </div>
    </div>
  );
}

function getRouteKind(pathname: string) {
  if (pathname === "/") {
    return "dashboard";
  }

  if (pathname === "/conversations" || pathname === "/tickets") {
    return "list";
  }

  if (pathname.startsWith("/conversations/") || pathname.startsWith("/tickets/")) {
    return "detail";
  }

  if (pathname.startsWith("/settings")) {
    return "settings";
  }

  return "default";
}

function prefersReducedMotion() {
  return (
    typeof window !== "undefined" &&
    "matchMedia" in window &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches
  );
}
