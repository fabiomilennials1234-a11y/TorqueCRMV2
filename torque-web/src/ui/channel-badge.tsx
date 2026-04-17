import {
  Facebook,
  Headphones,
  Instagram,
  MessageCircle,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

/* ─── Props ─── */
export interface ChannelBadgeProps {
  channel: "whatsapp" | "messenger" | "instagram" | "sz_chat";
  variant?: "icon-only" | "labeled" | "dot";
  size?: "sm" | "md";
  muted?: boolean;
  className?: string;
}

/* ─── Channel metadata ─── */
const CHANNELS: Record<
  ChannelBadgeProps["channel"],
  { label: string; icon: LucideIcon; bg: string; text: string; dot: string }
> = {
  whatsapp: {
    label: "WhatsApp",
    icon: MessageCircle,
    bg: "bg-channel-whatsapp/15",
    text: "text-channel-whatsapp",
    dot: "bg-channel-whatsapp",
  },
  messenger: {
    label: "Messenger",
    icon: Facebook,
    bg: "bg-channel-messenger/15",
    text: "text-channel-messenger",
    dot: "bg-channel-messenger",
  },
  instagram: {
    label: "Instagram",
    icon: Instagram,
    bg: "bg-channel-instagram/15",
    text: "text-channel-instagram",
    dot: "bg-channel-instagram",
  },
  sz_chat: {
    label: "SZ Chat",
    icon: Headphones,
    bg: "bg-channel-sz/15",
    text: "text-channel-sz",
    dot: "bg-channel-sz",
  },
};

/* ─── Size tokens ─── */
const ICON_SIZES = {
  sm: { wrapper: "h-6 w-6", icon: "h-3 w-3" },
  md: { wrapper: "h-8 w-8", icon: "h-4 w-4" },
} as const;

/* ─── Icon-only variant ─── */
function IconOnly({
  channel,
  size = "md",
}: Pick<ChannelBadgeProps, "channel" | "size">) {
  const meta = CHANNELS[channel];
  const s = ICON_SIZES[size ?? "md"];
  const Icon = meta.icon;

  return (
    <span
      className={cn(
        "inline-flex items-center justify-center rounded-full",
        meta.bg,
        meta.text,
        s.wrapper,
      )}
    >
      <Icon className={s.icon} strokeWidth={1.75} />
    </span>
  );
}

/* ─── Labeled variant ─── */
function Labeled({
  channel,
}: Pick<ChannelBadgeProps, "channel">) {
  const meta = CHANNELS[channel];
  const Icon = meta.icon;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5",
        meta.bg,
        meta.text,
      )}
    >
      <Icon className="h-3 w-3" strokeWidth={1.75} />
      <span className="text-2xs font-medium">{meta.label}</span>
    </span>
  );
}

/* ─── Dot variant ─── */
function Dot({ channel }: Pick<ChannelBadgeProps, "channel">) {
  const meta = CHANNELS[channel];

  return (
    <span
      className={cn(
        "inline-block h-1.5 w-1.5 rounded-full",
        meta.dot,
      )}
    />
  );
}

/* ─── Exported component ─── */
export function ChannelBadge({
  channel,
  variant = "icon-only",
  size = "md",
  muted = false,
  className,
}: ChannelBadgeProps) {
  const meta = CHANNELS[channel];

  return (
    <span
      aria-label={meta.label}
      className={cn(
        "inline-flex",
        muted && "pointer-events-none opacity-40",
        className,
      )}
    >
      {variant === "icon-only" && <IconOnly channel={channel} size={size} />}
      {variant === "labeled" && <Labeled channel={channel} />}
      {variant === "dot" && <Dot channel={channel} />}
    </span>
  );
}
