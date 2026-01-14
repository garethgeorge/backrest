import React, { useState } from "react";
import { Button, ButtonProps } from "../ui/button";

export const SpinButton = React.forwardRef<
  HTMLButtonElement,
  Omit<ButtonProps, "type" | "size"> & {
    onClickAsync: () => Promise<void>;
    type?: string;
    danger?: boolean;
    size?: string;
  }
>(
  (
    { onClickAsync, onClick: _onClick, type, variant, danger, size, ...props },
    ref,
  ) => {
    const [loading, setLoading] = useState(false);

    const onClick = async (e: React.MouseEvent<HTMLButtonElement>) => {
      if (loading) {
        return;
      }
      try {
        setLoading(true);
        await onClickAsync();
      } finally {
        setLoading(false);
      }
    };

    let mappedVariant = variant;
    if (!variant) {
      if (type === "primary")
        mappedVariant = "subtle"; // or solid? Backrest uses blue for primary? Chakra default solid is usually black/white. Subtle might be better or solid with colorPalette.
      else if (type === "default") mappedVariant = "outline";
      else if (type === "text" || type === "link") mappedVariant = "ghost";
      else if (type === "dashed") mappedVariant = "outline";
    }

    // AntD uses type="primary" implies blue/branded.
    // danger implies red
    let colorPalette = props.colorPalette;
    if (danger) colorPalette = "red";
    else if (type === "primary") colorPalette = "blue";

    // Determine HTML type
    const htmlType =
      type === "submit" || type === "reset" || type === "button"
        ? (type as "submit" | "reset" | "button")
        : "button";

    // Map legacy size
    let mappedSize = size;
    if (size === "small") mappedSize = "sm";
    else if (size === "large") mappedSize = "lg";
    else if (size === "middle") mappedSize = "md";

    return (
      <Button
        {...props}
        ref={ref}
        loading={loading}
        onClick={onClick}
        variant={mappedVariant}
        colorPalette={colorPalette}
        type={htmlType}
        size={mappedSize as any}
      />
    );
  },
);

SpinButton.displayName = "SpinButton";

export const ConfirmButton = React.forwardRef<
  HTMLButtonElement,
  Omit<ButtonProps, "type" | "size"> & {
    onClickAsync: () => Promise<void>;
    confirmTitle: React.ReactNode;
    confirmTimeout?: number; // milliseconds
    type?: string;
    danger?: boolean;
    size?: string;
  }
>(
  (
    {
      onClickAsync,
      confirmTimeout,
      confirmTitle,
      children,
      danger,
      size,
      ...props
    },
    ref,
  ) => {
    const [clicked, setClicked] = useState(false);

    if (confirmTimeout === undefined) {
      confirmTimeout = 2000;
    }

    const onClick = async () => {
      if (!clicked) {
        setClicked(true);
        setTimeout(() => {
          setClicked(false);
        }, confirmTimeout);
        return;
      }

      setClicked(false);
      await onClickAsync();
    };

    return (
      <SpinButton
        {...props}
        ref={ref}
        onClickAsync={onClick}
        danger={danger}
        size={size}
      >
        {clicked ? confirmTitle : children}
      </SpinButton>
    );
  },
);

ConfirmButton.displayName = "ConfirmButton";
