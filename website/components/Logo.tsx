import Link from "next/link";
import Image from "next/image";
import { cn } from "@/lib/utils";

type LogoProps = {
  size?: "sm" | "md" | "lg";
  href?: string | null;
  className?: string;
  /**
   * Only the header logo is above the fold. Passing `priority` on every copy
   * (the footer's included) makes Next preload the same image twice and
   * competes with the real LCP element for early bandwidth.
   */
  priority?: boolean;
};

const SIZE_MAP = {
  sm: { mark: 22, text: "text-base" },
  md: { mark: 28, text: "text-xl" },
  lg: { mark: 36, text: "text-2xl" },
};

/**
 * The mark used to be a remote `vntros_connect.svg` on an asset CDN —
 * an SVG whose entire content was one 3951x4096 base64 PNG, 3.45 MB over the
 * wire for a 28px glyph. It was `priority`, so it preloaded ahead of the hero
 * and single-handedly dominated LCP.
 *
 * It is now a local 256px raster, which is the honest format for what the
 * artwork actually is (a gradient 3D render, not vector geometry). Being local
 * means next/image resizes it to the requested density, serves AVIF/WebP, and
 * caches the derivative immutably — the request drops from 3.45 MB to a few kB
 * and costs no extra origin.
 */
export default function Logo({
  size = "md",
  href = "/",
  className,
  priority = false,
}: LogoProps) {
  const s = SIZE_MAP[size];
  const inner = (
    <span
      className={cn(
        "inline-flex items-center gap-2 font-semibold select-none",
        s.text,
        className,
      )}
    >
      <Image
        src="/logo-mark.png"
        alt=""
        aria-hidden="true"
        width={s.mark}
        height={s.mark}
        priority={priority}
        // No `sizes`. Passing it switches next/image to a responsive srcset
        // built from `deviceSizes`, which for a fixed 28px mark meant offering
        // the browser candidates up to 3840w. Omitting it keeps the fixed-size
        // behaviour: a 1x/2x srcset off `width`, which is all a 28px glyph can
        // ever use.
        className="object-contain"
      />
      <span className="text-vn-text">
        Vny<span className="text-vn-accent italic">tros</span>
      </span>
    </span>
  );

  if (!href) return inner;
  return <Link href={href}>{inner}</Link>;
}
