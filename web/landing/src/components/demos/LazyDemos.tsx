import { lazy, Suspense } from "react";

const BotChatDemo = lazy(() =>
  import("@/components/demos/BotChatDemo").then((m) => ({
    default: m.BotChatDemo,
  })),
);
const CaseDirDemo = lazy(() =>
  import("@/components/demos/CaseDirDemo").then((m) => ({
    default: m.CaseDirDemo,
  })),
);
const AdminDemo = lazy(() =>
  import("@/components/demos/AdminDemo").then((m) => ({
    default: m.AdminDemo,
  })),
);

function Skeleton() {
  return (
    <div className="aspect-[4/3] w-full animate-pulse rounded-xl bg-muted" />
  );
}

export function LazyBotChat() {
  return (
    <Suspense fallback={<Skeleton />}>
      <BotChatDemo />
    </Suspense>
  );
}

export function LazyCaseDir() {
  return (
    <Suspense fallback={<Skeleton />}>
      <CaseDirDemo />
    </Suspense>
  );
}

export function LazyAdmin() {
  return (
    <Suspense fallback={<Skeleton />}>
      <AdminDemo />
    </Suspense>
  );
}
