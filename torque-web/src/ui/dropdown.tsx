import * as RD from "@radix-ui/react-dropdown-menu";
import type { ComponentPropsWithoutRef } from "react";
import { Check, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

export const DropdownMenu = RD.Root;
export const DropdownMenuTrigger = RD.Trigger;
export const DropdownMenuGroup = RD.Group;
export const DropdownMenuPortal = RD.Portal;
export const DropdownMenuSub = RD.Sub;
export const DropdownMenuRadioGroup = RD.RadioGroup;

export function DropdownMenuContent({
  className,
  sideOffset = 6,
  ...props
}: ComponentPropsWithoutRef<typeof RD.Content>) {
  return (
    <RD.Portal>
      <RD.Content
        sideOffset={sideOffset}
        className={cn(
          "z-50 min-w-[12rem] overflow-hidden rounded-md bg-elevated/90 backdrop-blur-xl p-1",
          "shadow-elev-3 shadow-hairline",
          "data-[state=open]:animate-scale-in",
          className,
        )}
        {...props}
      />
    </RD.Portal>
  );
}

export function DropdownMenuItem({
  className,
  inset,
  ...props
}: ComponentPropsWithoutRef<typeof RD.Item> & { inset?: boolean }) {
  return (
    <RD.Item
      className={cn(
        "relative flex cursor-pointer select-none items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-ink-muted",
        "outline-none data-[highlighted]:bg-elevated/80 data-[highlighted]:text-ink",
        "data-[disabled]:pointer-events-none data-[disabled]:opacity-40",
        inset && "pl-8",
        className,
      )}
      {...props}
    />
  );
}

export function DropdownMenuCheckboxItem({
  className,
  children,
  checked,
  ...props
}: ComponentPropsWithoutRef<typeof RD.CheckboxItem>) {
  return (
    <RD.CheckboxItem
      checked={checked}
      className={cn(
        "relative flex cursor-pointer select-none items-center gap-2 rounded-sm py-1.5 pl-7 pr-2 text-sm text-ink-muted",
        "outline-none data-[highlighted]:bg-elevated/80 data-[highlighted]:text-ink",
        className,
      )}
      {...props}
    >
      <span className="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
        <RD.ItemIndicator>
          <Check className="h-3 w-3 text-accent" />
        </RD.ItemIndicator>
      </span>
      {children}
    </RD.CheckboxItem>
  );
}

export function DropdownMenuLabel({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof RD.Label>) {
  return (
    <RD.Label
      className={cn(
        "px-2 pb-1 pt-2 text-2xs uppercase tracking-[0.1em] text-ink-dim",
        className,
      )}
      {...props}
    />
  );
}

export function DropdownMenuSeparator({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof RD.Separator>) {
  return (
    <RD.Separator className={cn("-mx-1 my-1 h-px bg-hairline", className)} {...props} />
  );
}

export function DropdownMenuShortcut({ className, ...props }: React.HTMLAttributes<HTMLSpanElement>) {
  return (
    <span
      className={cn(
        "ml-auto font-mono text-2xs text-ink-dim tracking-wider",
        className,
      )}
      {...props}
    />
  );
}

export function DropdownMenuSubTrigger({
  className,
  inset,
  children,
  ...props
}: ComponentPropsWithoutRef<typeof RD.SubTrigger> & { inset?: boolean }) {
  return (
    <RD.SubTrigger
      className={cn(
        "flex cursor-pointer select-none items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-ink-muted",
        "outline-none data-[highlighted]:bg-elevated/80 data-[highlighted]:text-ink",
        inset && "pl-8",
        className,
      )}
      {...props}
    >
      {children}
      <ChevronRight className="ml-auto h-4 w-4" />
    </RD.SubTrigger>
  );
}

export function DropdownMenuSubContent({
  className,
  ...props
}: ComponentPropsWithoutRef<typeof RD.SubContent>) {
  return (
    <RD.SubContent
      className={cn(
        "z-50 min-w-[10rem] overflow-hidden rounded-md bg-elevated p-1 shadow-elev-3 shadow-hairline",
        "data-[state=open]:animate-scale-in",
        className,
      )}
      {...props}
    />
  );
}
