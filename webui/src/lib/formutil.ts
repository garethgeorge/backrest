import { FormInstance } from "antd";

export const validateForm = async <T>(form: FormInstance<T>) => {
  try {
    await form.validateFields();
    return form.getFieldsValue();
  } catch (e: any) {
    if (e.errorFields) {
      const firstError = (e as any).errorFields?.[0]
        ?.errors?.[0];
      throw new Error(firstError);
    }
    throw e;
  }
};

// regex allows alphanumeric, underscore, dash, and dot
// this should be kept in sync with values permitted by SanitizeID on the backend
export const namePattern = /^[a-zA-Z0-9_\-\.]+$/;
