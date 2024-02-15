import React from "react";
import { Button, ButtonProps } from "antd";
import { useState } from "react";

export const SpinButton: React.FC<ButtonProps & {
  onClickAsync: () => Promise<void>;
}> = ({ onClickAsync, ...props }) => {
  const [loading, setLoading] = useState(false);

  const onClick = async () => {
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

  return (
    <Button
      {...props}
      loading={loading}
      onClick={onClick}
    />
  );
}

export const ConfirmButton: React.FC<ButtonProps & {
  onClickAsync: () => Promise<void>;
  confirmTitle: React.ReactNode;
  confirmTimeout?: number; // milliseconds
}> = ({ onClickAsync, confirmTimeout, confirmTitle, children, ...props }) => {
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
      onClickAsync={onClick}
    >
      {clicked ? confirmTitle : children}
    </SpinButton>
  );
}
