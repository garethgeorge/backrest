import React, { useContext, useEffect, useMemo } from "react";
import { Form } from "antd";
import { clone, create, DescMessage, Message, MessageShape } from "@bufbuild/protobuf";
import { RepoSchema } from "../../../gen/ts/v1/config_pb";

type Paths<T> = T extends object
  ? {
      [K in keyof T]: `${Exclude<K, symbol>}${"" | `.${Paths<T[K]>}`}`;
    }[keyof T]
  : never;

type DeepIndex<T, K extends string> = T extends object
  ? K extends `${infer F}.${infer R}`
    ? DeepIndex<Idx<T, F>, R>
    : Idx<T, K>
  : never;

type Idx<T, K extends string> = K extends keyof T ? T[K] : never;

const getValue = <T extends {}, K extends Paths<T>>(
  obj: T,
  path: K
): DeepIndex<T, K> => {
  let curr: any = obj;
  const parts = path.split(".");
  for (let i = 0; i < parts.length; i++) {
    curr = curr[parts[i]];
  }
  return curr as DeepIndex<T, K>;
};

const setValue = <T extends {}, K extends Paths<T>>(
  obj: T,
  path: K,
  value: DeepIndex<T, K>
): void => {
  let curr: any = obj;
  const parts = path.split(".");
  for (let i = 0; i < parts.length - 1; i++) {
    curr = curr[parts[i]];
  }
  curr[parts[parts.length - 1]] = value;
};

interface typedFormCtx {
  formData: any;
  setFormData: (value: any) => any;
}

const TypedFormCtx = React.createContext<typedFormCtx>({
  formData: {},
  setFormData: () => {},
});

export const TypedForm = <Desc extends DescMessage>({
  schema,
  initialValue,
  props,
  children,
  onChange,
}: {
  props: any;
  schema: Desc;
  initialValue: MessageShape<Desc>;
  children: React.ReactNode;
  onChange?: (value: MessageShape<Desc>) => void;
}) => {
  const [form] = Form.useForm();
  const [formData, setFormData] =
    React.useState<MessageShape<Desc>>(initialValue);

  return (
    <TypedFormCtx.Provider value={{ formData, setFormData }}>
      <Form
        autoComplete="off"
        form={form}
        onFieldsChange={(changedFields, allFields) => {
          setFormData((prev) => {
            const next = clone(schema, prev);
            changedFields.forEach((field) => {
              setValue(next, field.name, field.value);
            });
            onChange?.(next);
            return next;
          });
        }}
        {...props}
      >
        {children}
      </Form>
    </TypedFormCtx.Provider>
  );
};

export const TypedFormItem = <T extends Message>({
  name,
  children,
  ...props,
}: {
  name: Paths<T>;
  children: React.ReactNode;
  props: any;
}): React.ReactElement => {
  const { formData } = useContext(TypedFormCtx);

  return (
    <Form.Item name={name as string} initialValue={getValue(formData, name)} {...props}>
      {children}
    </Form.Item>
  );
};

export const TypedFormOneof = <T extends Message>({
  name,
  children,
  ...props,
}: {
  name: Paths<T>;
  children: React.ReactNode;
  props: any;
}): React.ReactElement => {
  return <></>;
};

const const TypedFormOneofCase = <T extends Message>({
  name,
  children,
  ...props,
}: {
  name: Paths<T>;
  children: React.ReactNode;
  props: any;
}): React.ReactElement => {
  const { formData, setFormData } = useContext(TypedFormCtx);

  return <></>
}

const TestElement = () => {
  return (
    <TypedForm
      schema={RepoSchema}
      initialValue={create(RepoSchema, {})}
      onChange={onChange}
    >
      <TypedFormItem name="name" props={props}>
        <Input />
      </TypedFormItem>
      <TypedFormItem name="age" props={props}>
        <Input />
      </TypedFormItem>
    </TypedForm>
  );
}