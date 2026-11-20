using System;
using System.Drawing;
using System.Runtime.InteropServices;
using System.Windows.Forms;

namespace Aidex.ControlCenter
{
    public sealed class ActionButton : Button
    {
        private static readonly Color DisabledBackground = Color.FromArgb(232, 236, 234);
        private static readonly Color DisabledText = Color.FromArgb(77, 88, 84);
        private static readonly Color DisabledBorder = Color.FromArgb(204, 213, 209);

        protected override void OnEnabledChanged(EventArgs e)
        {
            base.OnEnabledChanged(e);
            Invalidate();
        }

        protected override void OnPaint(PaintEventArgs e)
        {
            if (Enabled)
            {
                base.OnPaint(e);
                return;
            }

            e.Graphics.Clear(DisabledBackground);
            using (var border = new Pen(DisabledBorder))
                e.Graphics.DrawRectangle(border, 0, 0, Width - 1, Height - 1);
            TextRenderer.DrawText(e.Graphics, Text, Font, ClientRectangle, DisabledText,
                TextFormatFlags.HorizontalCenter | TextFormatFlags.VerticalCenter | TextFormatFlags.EndEllipsis);
        }
    }

    // Give Explorer the right icon and relaunch target for a window hosted by PowerShell.
    public sealed class BrandedForm : Form
    {
        private static readonly Guid AppModelKey = new Guid("9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3");
        private static readonly int[] WindowKeys = { 2, 3, 4, 5 };
        private readonly string launcher;
        private readonly string iconPath;
        private readonly Icon brandIcon;
        private IPropertyStore properties;

        public BrandedForm(string launcher, string iconPath)
        {
            this.launcher = launcher;
            this.iconPath = iconPath;
            brandIcon = new Icon(iconPath, 64, 64);
            Icon = brandIcon;
        }

        protected override void OnHandleCreated(EventArgs e)
        {
            base.OnHandleCreated(e);
            Guid iid = typeof(IPropertyStore).GUID;
            if (SHGetPropertyStoreForWindow(Handle, ref iid, out properties) < 0)
                return;
            SetWindowProperty(2, "\"" + launcher + "\" --control-center");
            SetWindowProperty(3, iconPath + ",0");
            SetWindowProperty(4, "Centro de Control de AIDEX");
            SetWindowProperty(5, "AIDEX.ControlCenter");
        }

        protected override void OnHandleDestroyed(EventArgs e)
        {
            if (properties != null)
            {
                foreach (int key in WindowKeys)
                    SetWindowProperty(key, null);
                Marshal.ReleaseComObject(properties);
                properties = null;
            }
            base.OnHandleDestroyed(e);
        }

        protected override void Dispose(bool disposing)
        {
            base.Dispose(disposing);
            if (disposing)
                brandIcon.Dispose();
        }

        private void SetWindowProperty(int number, string text)
        {
            var key = new PropertyKey { Format = AppModelKey, Id = number };
            var value = new PropVariant();
            if (text != null)
            {
                value.Type = 31; // VT_LPWSTR
                value.Pointer = Marshal.StringToCoTaskMemUni(text);
            }
            try { properties.SetValue(ref key, ref value); }
            finally { if (value.Pointer != IntPtr.Zero) Marshal.FreeCoTaskMem(value.Pointer); }
        }

        [StructLayout(LayoutKind.Sequential)]
        private struct PropertyKey { public Guid Format; public int Id; }

        [StructLayout(LayoutKind.Sequential)]
        private struct PropVariant
        {
            public ushort Type, Reserved1, Reserved2, Reserved3;
            public IntPtr Pointer, Padding;
        }

        [ComImport, Guid("886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
        private interface IPropertyStore
        {
            [PreserveSig] int GetCount(out uint count);
            [PreserveSig] int GetAt(uint index, out PropertyKey key);
            [PreserveSig] int GetValue(ref PropertyKey key, out PropVariant value);
            [PreserveSig] int SetValue(ref PropertyKey key, ref PropVariant value);
            [PreserveSig] int Commit();
        }

        [DllImport("shell32.dll", ExactSpelling = true)]
        private static extern int SHGetPropertyStoreForWindow(IntPtr window, ref Guid iid, out IPropertyStore store);
    }
}
