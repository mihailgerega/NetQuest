import Link from "next/link";
import { cn } from "@/lib/utils";

type ButtonProps = {
  href?: string;
  children: React.ReactNode;
  variant?: "primary" | "secondary" | "ghost";
  className?: string;
} & React.ButtonHTMLAttributes<HTMLButtonElement>;

export function Button({ href, children, variant = "primary", className, type = "button", ...buttonProps }: ButtonProps) {
  const classes = cn(
    "inline-flex min-h-11 items-center justify-center rounded-md px-5 text-sm font-semibold transition focus:outline-none focus:ring-2 focus:ring-signal-cyan focus:ring-offset-2 focus:ring-offset-ink-950",
    variant === "primary" && "bg-signal-cyan text-ink-950 shadow-glow hover:bg-sky-300 disabled:cursor-not-allowed disabled:opacity-55",
    variant === "secondary" && "border border-white/[0.14] bg-white/[0.07] text-white hover:bg-white/[0.12]",
    variant === "ghost" && "text-slate-200 hover:bg-white/[0.08]",
    className
  );

  if (href) {
    return (
      <Link className={classes} href={href}>
        {children}
      </Link>
    );
  }

  return (
    <button className={classes} type={type} {...buttonProps}>
      {children}
    </button>
  );
}
