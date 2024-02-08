import { FormInstance } from "antd";
import type { ValidateErrorEntity } from "rc-field-form/lib/interface";

export const validateForm = async <T>(form: FormInstance<T>) => {
  try {
    await form.validateFields();
    return form.getFieldsValue();
  } catch (e: any) {
    if (e.errorFields) {
      const firstError = (e as ValidateErrorEntity).errorFields?.[0]
        ?.errors?.[0];
      throw new Error(firstError);
    }
    throw e;
  }
};

// regex allows alphanumeric, underscore, dash, and dot
export const namePattern = /^[a-zA-Z0-9_\-\.]+$/;
