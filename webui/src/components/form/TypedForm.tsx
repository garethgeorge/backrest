import React, { useContext, useEffect, useMemo, useState } from "react";
import { Form } from "antd";
import {
  clone,
  create,
  DescMessage,
  Message,
  MessageShape,
} from "@bufbuild/protobuf";
import { useWatch } from "antd/es/form/Form";
import useFormInstance from "antd/es/form/hooks/useFormInstance";

export type Paths<T> = T extends object
  ? {
      [K in Exclude<keyof T, symbol>]: `${K}${Paths<T[K]> extends never
        ? ""
        : `.${Paths<T[K]>}`}`;
    }[Exclude<keyof T, symbol>]
  : never;

export type DeepIndex<T, K extends string> = T extends object
  ? K extends `${infer F}.${infer R}`
    ? DeepIndex<Idx<T, F>, R>
    : Idx<T, K>
  : never;

type Idx<T, K extends string> = K extends keyof T ? T[K] : never;

const getValue = (obj: any, path: string): any => {
  let curr: any = obj;
  const parts = path.split(".");
  for (let i = 0; i < parts.length; i++) {
    if (Array.isArray(curr)) {
      curr = curr[parseInt(parts[i])];
    } else {
      curr = curr[parts[i]];
    }
  }
  return curr as any;
};

const setValue = (obj: any, path: string, value: any): void => {
  let curr: any = obj;
  const parts = path.split(".");
  for (let i = 0; i < parts.length - 1; i++) {
    if (Array.isArray(curr)) {
      curr = curr[parseInt(parts[i])];
    } else {
      curr = curr[parts[i]];
    }
  }
  curr[parts[parts.length - 1]] = value;
};

interface typedFormCtx {
  formData: Message;
  schema: DescMessage;
  setFormData: (fn: (prev: any) => any) => void;
}

const TypedFormCtx = React.createContext<typedFormCtx | null>(null);
const FieldPrefixCtx = React.createContext<string>("");

export const TypedForm = <T extends Message>({
  schema,
  formData,
  setFormData,
  children,
  ...props
}: {
  schema: DescMessage;
  formData: T;
  setFormData: (fn: (prev: T) => T) => void;
  children?: React.ReactNode;
} & {
  [key: string]: any;
}) => {
  const [form] = Form.useForm<T>();

  return (
    <TypedFormCtx.Provider value={{ formData, setFormData, schema }}>
      <Form autoComplete="off" form={form} {...props}>
        {children}
      </Form>
    </TypedFormCtx.Provider>
  );
};

export const TypedFormItem = <T extends Message>({
  field,
  children,
  ...props
}: {
  field: Paths<T>;
  children?: React.ReactNode;
} & {
  [key: string]: any;
}): React.ReactElement => {
  const prefix = useContext(FieldPrefixCtx);
  const { formData, setFormData, schema } = useContext(TypedFormCtx)!;
  const form = useFormInstance();
  const resolvedField = prefix + field;

  const [lastSeen, setLastSeen] = useState<any>(null);
  const formValue = useWatch(resolvedField);
  const formDataStateValue = getValue(formData, resolvedField);

  useEffect(() => {
    if (lastSeen !== formValue) {
      setFormData((prev: any) => {
        const next = clone(schema, prev) as any as T;
        setValue(next, resolvedField, formValue);
        return next;
      });
      setLastSeen(formValue);
    } else if (lastSeen !== formDataStateValue) {
      form.setFieldsValue({ [resolvedField]: formDataStateValue } as any);
      setLastSeen(formDataStateValue);
    }
  }, [lastSeen, formValue, formDataStateValue]);

  return (
    <Form.Item
      name={resolvedField as string}
      initialValue={getValue(formData, field)}
      {...props}
    >
      {children}
    </Form.Item>
  );
};

// This is a helper component that sets a prefix for all of its children.
const WithFieldPrefix = ({
  prefix,
  children,
}: {
  prefix: string;
  children?: React.ReactNode;
}): React.ReactNode => {
  const prev = useContext(FieldPrefixCtx);
  return (
    <FieldPrefixCtx.Provider value={prev + prefix}>
      {children}
    </FieldPrefixCtx.Provider>
  );
};

export const TypedFormList = <T extends Message>({
  field,
  children,
  ...props
}: {
  field: Paths<T>;
  children?: React.ReactNode;
} & {
  [key: string]: any;
}): React.ReactElement => {
  return (
    <Form.List name={field} {...props}>
      <WithFieldPrefix prefix={field} children={children} />
    </Form.List>
  );
};

interface oneofCase<T extends {}, F extends Paths<T>> {
  case: DeepIndex<T, `${F}.case`>;
  create: () => DeepIndex<T, `${F}.value`>;
  view: React.ReactNode;
}

/*
export const TypedFormOneof = <T extends Message, F extends Paths<T>>({
  field,
  items,
}: {
  field: F;
  items: oneofCase<T, F>[];
}): React.ReactNode => {
  const { formData, setFormData, schema } = useContext(TypedFormCtx)!;
  const c = useWatch(`${field}.case`);
  useEffect(() => {
    for (const item of items) {
      if (item.case === c) {
        setFormData((prev: any) => {
          const next = clone(schema, prev as T) as T;
          setValue(next, field, {
            case: c,
            value: item.create(),
          } as any);
          return next;
        });
      }
    }
    throw new Error("case " + c + " not found");
  }, [c]);

  for (const item of items) {
    if (item.case === c) {
      return item.view;
    }
  }
  return null;
};

*/
